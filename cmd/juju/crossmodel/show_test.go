// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package crossmodel_test

import (
	"github.com/juju/cmd"
	"github.com/juju/errors"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"
	"gopkg.in/juju/charm.v6-unstable"

	"github.com/juju/juju/apiserver/params"
	"github.com/juju/juju/cmd/juju/crossmodel"
	"github.com/juju/juju/testing"
)

type showSuite struct {
	BaseCrossModelSuite
	mockAPI *mockShowAPI
}

var _ = gc.Suite(&showSuite{})

func (s *showSuite) SetUpTest(c *gc.C) {
	s.BaseCrossModelSuite.SetUpTest(c)

	s.mockAPI = &mockShowAPI{
		serviceTag: "hosted-db2",
		desc:       "IBM DB2 Express Server Edition is an entry level database system",
	}
}

func (s *showSuite) runShow(c *gc.C, args ...string) (*cmd.Context, error) {
	return testing.RunCommand(c, crossmodel.NewShowEndpointsCommandForTest(s.mockAPI), args...)
}

func (s *showSuite) TestShowNoUrl(c *gc.C) {
	s.assertShowError(c, nil, ".*must specify endpoint URL.*")
}

func (s *showSuite) TestShowApiError(c *gc.C) {
	s.mockAPI.msg = "fail"
	s.assertShowError(c, []string{"local:/u/fred/db2"}, ".*fail.*")
}

func (s *showSuite) TestShowURLError(c *gc.C) {
	s.mockAPI.serviceTag = "invalid_tag"
	s.assertShowError(c, []string{"local:/u/fred/prod/foo/db2"}, ".*invalid.*")
}

func (s *showSuite) TestShowYaml(c *gc.C) {
	s.assertShow(
		c,
		[]string{"local:/u/fred/db2", "--format", "yaml"},
		`
hosted-db2:
  endpoints:
    db2:
      interface: hhtp
      role: requirer
    log:
      interface: http
      role: provider
  description: IBM DB2 Express Server Edition is an entry level database system
`[1:],
	)
}

func (s *showSuite) TestShowTabular(c *gc.C) {
	s.assertShow(
		c,
		[]string{"local:/u/fred/db2", "--format", "tabular"},
		`
SERVICE     DESCRIPTION                     ENDPOINT  INTERFACE  ROLE
hosted-db2  IBM DB2 Express Server Edition  db2       hhtp       requirer
            an entry level database system  log       http       provider
                                                                 

`[1:],
	)
}

func (s *showSuite) TestShowTabularExactly100Desc(c *gc.C) {
	s.mockAPI.desc = s.mockAPI.desc + s.mockAPI.desc[:36]
	s.assertShow(
		c,
		[]string{"local:/u/fred/db2", "--format", "tabular"},
		`
SERVICE     DESCRIPTION                     ENDPOINT  INTERFACE  ROLE
hosted-db2  IBM DB2 Express Server Edition  db2       hhtp       requirer
            an entry level database         log       http       provider
            DB2 Express Server Edition is                        
                                                                 

`[1:],
	)
}

func (s *showSuite) TestShowTabularMoreThan100Desc(c *gc.C) {
	s.mockAPI.desc = s.mockAPI.desc + s.mockAPI.desc
	s.assertShow(
		c,
		[]string{"local:/u/fred/db2", "--format", "tabular"},
		`
SERVICE     DESCRIPTION                     ENDPOINT  INTERFACE  ROLE
hosted-db2  IBM DB2 Express Server Edition  db2       hhtp       requirer
            an entry level database         log       http       provider
            DB2 Express Server Edition                           
                                                                 

`[1:],
	)
}

func (s *showSuite) assertShow(c *gc.C, args []string, expected string) {
	context, err := s.runShow(c, args...)
	c.Assert(err, jc.ErrorIsNil)

	obtained := testing.Stdout(context)
	c.Assert(obtained, gc.Matches, expected)
}

func (s *showSuite) assertShowError(c *gc.C, args []string, expected string) {
	_, err := s.runShow(c, args...)
	c.Assert(err, gc.ErrorMatches, expected)
}

type mockShowAPI struct {
	msg, serviceTag, desc string
}

func (s mockShowAPI) Close() error {
	return nil
}

func (s mockShowAPI) ServiceOffer(url string) (params.ServiceOffer, error) {
	if s.msg != "" {
		return params.ServiceOffer{}, errors.New(s.msg)
	}

	return params.ServiceOffer{
		ServiceName:        s.serviceTag,
		ServiceDescription: s.desc,
		Endpoints: []params.RemoteEndpoint{
			params.RemoteEndpoint{Name: "log", Interface: "http", Role: charm.RoleProvider},
			params.RemoteEndpoint{Name: "db2", Interface: "hhtp", Role: charm.RoleRequirer},
		},
	}, nil
}
