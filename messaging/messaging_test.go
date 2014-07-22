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

package messaging

import (
	"time"

	. "launchpad.net/gocheck"
	"testing"

	"launchpad.net/ubuntu-push/click"
	clickhelp "launchpad.net/ubuntu-push/click/testing"
	"launchpad.net/ubuntu-push/launch_helper"
	"launchpad.net/ubuntu-push/messaging/cmessaging"
	helpers "launchpad.net/ubuntu-push/testing"
)

// hook up gocheck
func Test(t *testing.T) { TestingT(t) }

type MessagingSuite struct {
	log *helpers.TestLogger
	app *click.AppId
}

var _ = Suite(&MessagingSuite{})

func (ms *MessagingSuite) SetUpSuite(c *C) {
	cAddNotification = func(a string, n string, c *launch_helper.Card, payload *cmessaging.Payload) {
		ms.log.Debugf("ADD: app: %s, not: %s, card: %v, chan: %v", a, n, c, payload)
	}
}

func (ms *MessagingSuite) SetUpTest(c *C) {
	ms.log = helpers.NewTestLogger(c, "debug")
	ms.app = clickhelp.MustParseAppId("com.example.test_test_0")
}

func (ms *MessagingSuite) TestPresentPresents(c *C) {
	mmu := New(ms.log)
	card := launch_helper.Card{Summary: "ehlo", Persist: true}
	notif := launch_helper.Notification{Card: &card}

	c.Check(mmu.Present(ms.app, "notif-id", &notif), Equals, true)

	c.Check(ms.log.Captured(), Matches, `(?s).* ADD:.*notif-id.*`)
}

func (ms *MessagingSuite) TestPresentDoesNotPresentsIfNoSummary(c *C) {
	mmu := New(ms.log)
	card := launch_helper.Card{Persist: true}
	notif := launch_helper.Notification{Card: &card}

	c.Check(mmu.Present(ms.app, "notif-id", &notif), Equals, false)

	c.Check(ms.log.Captured(), Matches, "(?sm).*has no persistable card.*")
}

func (ms *MessagingSuite) TestPresentDoesNotPresentsIfNotPersist(c *C) {
	mmu := New(ms.log)
	card := launch_helper.Card{Summary: "ehlo"}
	notif := launch_helper.Notification{Card: &card}

	c.Check(mmu.Present(ms.app, "notif-id", &notif), Equals, false)

	c.Check(ms.log.Captured(), Matches, "(?sm).*has no persistable card.*")
}

func (ms *MessagingSuite) TestPresentPanicsIfNil(c *C) {
	mmu := New(ms.log)
	c.Check(func() { mmu.Present(ms.app, "notif-id", nil) }, Panics, `please check notification is not nil before calling present`)
}

func (ms *MessagingSuite) TestPresentDoesNotPresentsIfNilCard(c *C) {
	mmu := New(ms.log)
	c.Check(mmu.Present(ms.app, "notif-id", &launch_helper.Notification{}), Equals, false)
	c.Check(ms.log.Captured(), Matches, "(?sm).*no persistable card.*")
}

func (ms *MessagingSuite) TestPresentWithActions(c *C) {
	mmu := New(ms.log)
	card := launch_helper.Card{Summary: "ehlo", Persist: true, Actions: []string{"action-1"}}
	notif := launch_helper.Notification{Card: &card, Tag: "a-tag"}

	c.Check(mmu.Present(ms.app, "notif-id", &notif), Equals, true)

	c.Check(ms.log.Captured(), Matches, `(?s).* ADD:.*notif-id.*`)

	payload, _ := mmu.notifications["notif-id"]
	c.Check(payload.Ch, Equals, mmu.Ch)
	c.Check(len(payload.Actions), Equals, 2)
	c.Check(payload.Tag, Equals, "a-tag")
	rawAction := "{\"app\":\"com.example.test_test_0\",\"act\":\"action-1\",\"nid\":\"notif-id\"}"
	c.Check(payload.Actions[0], Equals, rawAction)
	c.Check(payload.Actions[1], Equals, "action-1")
}

func (ms *MessagingSuite) TestTagsListsTags(c *C) {
	mmu := New(ms.log)
	f := func(s string) *launch_helper.Notification {
		card := launch_helper.Card{Summary: s, Persist: true}
		return &launch_helper.Notification{Card: &card, Tag: s}
	}

	c.Check(mmu.Tags(ms.app), IsNil)
	c.Assert(mmu.Present(ms.app, "notif1", f("one")), Equals, true)
	c.Check(mmu.Tags(ms.app), DeepEquals, map[string][]string{"card": {"one"}})
	c.Assert(mmu.Present(ms.app, "notif2", f("two")), Equals, true)
	c.Check(mmu.Tags(ms.app), DeepEquals, map[string][]string{"card": {"one", "two"}})
	// and an empty notification doesn't count
	c.Assert(mmu.Present(ms.app, "notif2", &launch_helper.Notification{Tag: "xxx"}), Equals, false)
	c.Check(mmu.Tags(ms.app), DeepEquals, map[string][]string{"card": {"one", "two"}})
	// and they go away if we remove one
	mmu.RemoveNotification("notif1")
	c.Check(mmu.Tags(ms.app), DeepEquals, map[string][]string{"card": {"two"}})
	mmu.RemoveNotification("notif2")
	c.Check(mmu.Tags(ms.app), IsNil)
}

func (ms *MessagingSuite) TestRemoveNotification(c *C) {
	mmu := New(ms.log)
	card := launch_helper.Card{Summary: "ehlo", Persist: true, Actions: []string{"action-1"}}
	actions := []string{"{\"app\":\"com.example.test_test_0\",\"act\":\"action-1\",\"nid\":\"notif-id\"}", "action-1"}
	mmu.addNotification(ms.app, "notif-id", "a-tag", &card, actions)

	// check it's there
	payload, ok := mmu.notifications["notif-id"]
	c.Check(ok, Equals, true)
	c.Check(payload.Actions, DeepEquals, actions)
	c.Check(payload.Tag, Equals, "a-tag")
	c.Check(payload.Ch, Equals, mmu.Ch)
	// remove the notification
	mmu.RemoveNotification("notif-id")
	// check it's gone
	_, ok = mmu.notifications["notif-id"]
	c.Check(ok, Equals, false)
}

func (ms *MessagingSuite) TestCleanupStaleNotification(c *C) {
	mmu := New(ms.log)
	card := launch_helper.Card{Summary: "ehlo", Persist: true, Actions: []string{"action-1"}}
	actions := []string{"{\"app\":\"com.example.test_test_0\",\"act\":\"action-1\",\"nid\":\"notif-id\"}", "action-1"}
	mmu.addNotification(ms.app, "notif-id", "", &card, actions)

	// check it's there
	_, ok := mmu.notifications["notif-id"]
	c.Check(ok, Equals, true)

	// patch cnotificationexists to return true
	cNotificationExists = func(did string, nid string) bool {
		return true
	}
	// remove the notification
	mmu.cleanUpNotifications()
	// check it's still there
	_, ok = mmu.notifications["notif-id"]
	c.Check(ok, Equals, true)
	// patch cnotificationexists to return false
	cNotificationExists = func(did string, nid string) bool {
		return false
	}
	// remove the notification
	mmu.cleanUpNotifications()
	// check it's gone
	_, ok = mmu.notifications["notif-id"]
	c.Check(ok, Equals, false)
}

func (ms *MessagingSuite) TestCleanupLoop(c *C) {
	// make the cleanup loop run a bit faster
	cleanupLoopDuration = 100 * time.Nanosecond
	mmu := New(ms.log)
	// patch cnotificationexists to return true
	cNotificationExists = func(did string, nid string) bool {
		return true
	}
	card := launch_helper.Card{Summary: "ehlo", Persist: true, Actions: []string{"action-1"}}
	actions := []string{"{\"app\":\"com.example.test_test_0\",\"act\":\"action-1\",\"nid\":\"notif-id\"}", "action-1"}
	mmu.addNotification(ms.app, "notif-id", "", &card, actions)

	// check it's there
	_, ok := mmu.notifications["notif-id"]
	c.Check(ok, Equals, true)

	// statr the cleanup loop
	mmu.StartCleanupLoop()
	// patch cnotificationexists to return false
	cNotificationExists = func(did string, nid string) bool {
		return false
	}
	// wait for a couple of loops
	<-time.After(500 * time.Nanosecond)
	// check it's gone
	_, ok = mmu.notifications["notif-id"]
	c.Check(ok, Equals, false)

	// stop the loop and check that it's actually stopped.
	mmu.StopCleanupLoop()
	// wait for a couple of loops
	<-time.After(1 * time.Millisecond)
	mmu.addNotification(ms.app, "notif-id-1", "", &card, actions)
	// check it's there
	_, ok = mmu.notifications["notif-id-1"]
	c.Check(ok, Equals, true)
}
