/*
 Copyright 2013-2014 Canonical Ltd.

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

package main

import (
	"net"
	"net/http"
	"time"
)

// A HTTPServeConfig holds the HTTP server config.
type HTTPServeConfig interface {
	HTTPAddr() string
	HTTPReadTimeout() time.Duration
	HTTPWriteTimeout() time.Duration
}

// RunHTTPServe serves HTTP requests.
func RunHTTPServe(lst net.Listener, h http.Handler, cfg HTTPServeConfig) error {
	srv := &http.Server{
		Handler:      h,
		ReadTimeout:  cfg.HTTPReadTimeout(),
		WriteTimeout: cfg.HTTPReadTimeout(),
	}
	return srv.Serve(lst)
}
