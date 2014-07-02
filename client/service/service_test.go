/*
 Copyright 2014 Canonical Ltd.

 This program is free software: you can redistribute it and/or modify it
 under the terms of the GNU General Public License version 3, as published
 by the Free Software Foundation.

 This program is distributed in the hope that it will be useful, but
 WITHOUT ANY WARRANTY; without even the implied warranties of
 MERCHANTABILITY, SATISFACTORY QUALITY, or FITNESS FOR A PARTICULAR
 PURPOSE.  See the GNU General Public License for more details.

 You should have received a copy of the GNU General Public License along
 with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	. "launchpad.net/gocheck"

	"launchpad.net/ubuntu-push/bus"
	testibus "launchpad.net/ubuntu-push/bus/testing"
	"launchpad.net/ubuntu-push/logger"
	helpers "launchpad.net/ubuntu-push/testing"
	"launchpad.net/ubuntu-push/testing/condition"
)

func TestService(t *testing.T) { TestingT(t) }

type serviceSuite struct {
	log logger.Logger
	bus bus.Endpoint
}

var _ = Suite(&serviceSuite{})

func (ss *serviceSuite) SetUpTest(c *C) {
	ss.log = helpers.NewTestLogger(c, "debug")
	ss.bus = testibus.NewTestingEndpoint(condition.Work(true), nil)
}

func (ss *serviceSuite) TestStart(c *C) {
	svc := NewPushService(ss.bus, ss.log)
	c.Check(svc.IsRunning(), Equals, false)
	c.Check(svc.Start(), IsNil)
	c.Check(svc.IsRunning(), Equals, true)
	svc.Stop()
}

func (ss *serviceSuite) TestStartTwice(c *C) {
	svc := NewPushService(ss.bus, ss.log)
	c.Check(svc.Start(), IsNil)
	c.Check(svc.Start(), Equals, AlreadyStarted)
	svc.Stop()
}

func (ss *serviceSuite) TestStartNoLog(c *C) {
	svc := NewPushService(ss.bus, nil)
	c.Check(svc.Start(), Equals, NotConfigured)
}

func (ss *serviceSuite) TestStartNoBus(c *C) {
	svc := NewPushService(nil, ss.log)
	c.Check(svc.Start(), Equals, NotConfigured)
}

func (ss *serviceSuite) TestStartFailsOnBusDialFailure(c *C) {
	bus := testibus.NewTestingEndpoint(condition.Work(false), nil)
	svc := NewPushService(bus, ss.log)
	c.Check(svc.Start(), ErrorMatches, `.*(?i)cond said no.*`)
	svc.Stop()
}

func (ss *serviceSuite) TestStartGrabsName(c *C) {
	svc := NewPushService(ss.bus, ss.log)
	c.Assert(svc.Start(), IsNil)
	callArgs := testibus.GetCallArgs(ss.bus)
	defer svc.Stop()
	c.Assert(callArgs, NotNil)
	c.Check(callArgs[0].Member, Equals, "::GrabName")
}

func (ss *serviceSuite) TestStopClosesBus(c *C) {
	svc := NewPushService(ss.bus, ss.log)
	c.Assert(svc.Start(), IsNil)
	svc.Stop()
	callArgs := testibus.GetCallArgs(ss.bus)
	c.Assert(callArgs, NotNil)
	c.Check(callArgs[len(callArgs)-1].Member, Equals, "::Close")
}

// registration tests

func (ss *serviceSuite) TestSetRegURLWorks(c *C) {
	svc := NewPushService(ss.bus, ss.log)
	c.Check(svc.regURL, Equals, "")
	svc.SetRegistrationURL("xyzzy://")
	c.Check(svc.regURL, Equals, "xyzzy://")
}

func (ss *serviceSuite) TestSetAuthGetterWorks(c *C) {
	svc := NewPushService(ss.bus, ss.log)
	c.Check(svc.authGetter, IsNil)
	f := func(string) string { return "" }
	svc.SetAuthGetter(f)
	c.Check(fmt.Sprintf("%#v", svc.authGetter), Equals, fmt.Sprintf("%#v", f))
}

func (ss *serviceSuite) TestGetRegAuthWorks(c *C) {
	svc := NewPushService(ss.bus, ss.log)
	svc.SetRegistrationURL("xyzzy://")
	ch := make(chan string, 1)
	f := func(s string) string { ch <- s; return "Auth " + s }
	svc.SetAuthGetter(f)
	c.Check(svc.getRegistrationAuthorization(), Equals, "Auth xyzzy://")
	c.Assert(len(ch), Equals, 1)
	c.Check(<-ch, Equals, "xyzzy://")
}

func (ss *serviceSuite) TestGetRegAuthDoesNotPanic(c *C) {
	svc := NewPushService(ss.bus, ss.log)
	c.Check(svc.getRegistrationAuthorization(), Equals, "")
}

func (ss *serviceSuite) TestRegistrationFailsIfBadArgs(c *C) {
	reg, err := new(PushService).register("", []interface{}{1}, nil)
	c.Check(reg, IsNil)
	c.Check(err, Equals, BadArgCount)
}

func (ss *serviceSuite) TestRegistrationWorks(c *C) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 256)
		n, e := r.Body.Read(buf)
		c.Assert(e, IsNil)
		req := registrationRequest{}
		c.Assert(json.Unmarshal(buf[:n], &req), IsNil)
		c.Check(req, DeepEquals, registrationRequest{"fake-device-id", "an-app-id"})

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"ok":true,"token":"blob-of-bytes"}`)
	}))
	defer ts.Close()

	svc := NewPushService(ss.bus, ss.log)
	svc.SetAuthGetter(func(string) string { return "tok" })
	svc.SetRegistrationURL(ts.URL)
	svc.SetDeviceId("fake-device-id")
	// this'll check (un)quoting, too
	reg, err := svc.register("/an_2dapp_2did", nil, nil)
	c.Assert(err, IsNil)
	c.Assert(reg, HasLen, 1)
	regs, ok := reg[0].(string)
	c.Check(ok, Equals, true)
	c.Check(regs, Equals, "blob-of-bytes")
}

func (ss *serviceSuite) TestRegistrationOverrideWorks(c *C) {
	os.Setenv("PUSH_REG_stuff", "42")
	defer os.Setenv("PUSH_REG_stuff", "")

	reg, err := new(PushService).register("/stuff", nil, nil)
	c.Assert(reg, HasLen, 1)
	regs, ok := reg[0].(string)
	c.Check(ok, Equals, true)
	c.Check(regs, Equals, "42")
	c.Check(err, IsNil)
}

func (ss *serviceSuite) TestRegistrationFailsOnBadReqURL(c *C) {
	svc := NewPushService(ss.bus, ss.log)
	svc.SetRegistrationURL("%gh")
	reg, err := svc.register("thing", nil, nil)
	c.Check(reg, IsNil)
	c.Check(err, ErrorMatches, "unable to build register request: .*")
}

func (ss *serviceSuite) TestRegistrationFailsOnBadAuth(c *C) {
	svc := NewPushService(ss.bus, ss.log)
	// ... no auth added
	reg, err := svc.register("thing", nil, nil)
	c.Check(reg, IsNil)
	c.Check(err, Equals, BadAuth)
}

func (ss *serviceSuite) TestRegistrationFailsOnNoServer(c *C) {
	svc := NewPushService(ss.bus, ss.log)
	svc.SetRegistrationURL("xyzzy://")
	svc.SetAuthGetter(func(string) string { return "tok" })
	reg, err := svc.register("thing", nil, nil)
	c.Check(reg, IsNil)
	c.Check(err, ErrorMatches, "unable to request registration: .*")
}

func (ss *serviceSuite) TestRegistrationFailsOn40x(c *C) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "I'm a teapot", 418)
	}))
	defer ts.Close()

	svc := NewPushService(ss.bus, ss.log)
	svc.SetAuthGetter(func(string) string { return "tok" })
	svc.SetRegistrationURL(ts.URL)
	reg, err := svc.register("/thing", nil, nil)
	c.Check(err, Equals, BadRequest)
	c.Check(reg, IsNil)
}

func (ss *serviceSuite) TestRegistrationFailsOn50x(c *C) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Not implemented", 501)
	}))
	defer ts.Close()

	svc := NewPushService(ss.bus, ss.log)
	svc.SetAuthGetter(func(string) string { return "tok" })
	svc.SetRegistrationURL(ts.URL)
	reg, err := svc.register("/thing", nil, nil)
	c.Check(err, Equals, BadServer)
	c.Check(reg, IsNil)
}

func (ss *serviceSuite) TestRegistrationFailsOnBadJSON(c *C) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 256)
		n, e := r.Body.Read(buf)
		c.Assert(e, IsNil)
		req := registrationRequest{}
		c.Assert(json.Unmarshal(buf[:n], &req), IsNil)
		c.Check(req, DeepEquals, registrationRequest{"fake-device-id", "an-app-id"})

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{`)
	}))
	defer ts.Close()

	svc := NewPushService(ss.bus, ss.log)
	svc.SetAuthGetter(func(string) string { return "tok" })
	svc.SetRegistrationURL(ts.URL)
	svc.SetDeviceId("fake-device-id")
	// this'll check (un)quoting, too
	reg, err := svc.register("/an_2dapp_2did", nil, nil)
	c.Check(reg, IsNil)
	c.Check(err, ErrorMatches, "unable to unmarshal register response: .*")
}

func (ss *serviceSuite) TestRegistrationFailsOnBadJSONDocument(c *C) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 256)
		n, e := r.Body.Read(buf)
		c.Assert(e, IsNil)
		req := registrationRequest{}
		c.Assert(json.Unmarshal(buf[:n], &req), IsNil)
		c.Check(req, DeepEquals, registrationRequest{"fake-device-id", "an-app-id"})

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"bananas": "very yes"}`)
	}))
	defer ts.Close()

	svc := NewPushService(ss.bus, ss.log)
	svc.SetAuthGetter(func(string) string { return "tok" })
	svc.SetRegistrationURL(ts.URL)
	svc.SetDeviceId("fake-device-id")
	// this'll check (un)quoting, too
	reg, err := svc.register("/an_2dapp_2did", nil, nil)
	c.Check(reg, IsNil)
	c.Check(err, Equals, BadToken)
}