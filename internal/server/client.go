package server

/*
 * AWS SSO CLI
 * Copyright (c) 2021-2022 Aaron Turner  <synfinatic at gmail dot com>
 *
 * This program is free software: you can redistribute it
 * and/or modify it under the terms of the GNU General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or with the authors permission any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/davecgh/go-spew/spew"
	"github.com/synfinatic/aws-sso-cli/internal/storage"
)

type ECSClient struct {
	port int
}

func NewECSClient(port int) *ECSClient {
	return &ECSClient{
		port: port,
	}
}

func (c *ECSClient) LoadUrl(profile string) string {
	if profile == "" {
		return fmt.Sprintf("http://localhost:%d/", c.port)
	}
	return fmt.Sprintf("http://localhost:%d%s/%s", c.port, SLOT_ROUTE, url.QueryEscape(profile))
}

func (c *ECSClient) ProfileUrl() string {
	return fmt.Sprintf("http://localhost:%d%s", c.port, PROFILE_ROUTE)
}

func (c *ECSClient) ListUrl() string {
	return fmt.Sprintf("http://localhost:%d%s", c.port, SLOT_ROUTE)
}

type ECSClientRequest struct {
	Creds       *storage.RoleCredentials `json:"Creds"`
	ProfileName string                   `json:"ProfileName"`
}

func (c *ECSClient) SubmitCreds(creds *storage.RoleCredentials, profile string, slotted bool) error {
	log.Debugf("loading %s in a slot: %v", profile, slotted)
	cr := ECSClientRequest{
		Creds:       creds,
		ProfileName: profile,
	}
	j, err := json.Marshal(cr)
	if err != nil {
		return err
	}
	var path string
	if slotted {
		path = profile
	}
	req, err := http.NewRequest(http.MethodPut, c.LoadUrl(path), bytes.NewBuffer(j))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", CHARSET_JSON)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	return CheckDoResponse(resp)
}

func (c *ECSClient) GetProfile() (ListProfilesResponse, error) {
	lpr := ListProfilesResponse{}
	client := &http.Client{}
	resp, err := client.Get(c.ProfileUrl())
	if err != nil {
		return lpr, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return lpr, err
	}

	if err = json.Unmarshal(body, &lpr); err != nil {
		return lpr, err
	}
	log.Debugf("resp: %s", spew.Sdump(lpr))

	return lpr, nil
}

// ListProfiles returns a list of profiles that are loaded into slots
func (c *ECSClient) ListProfiles() ([]ListProfilesResponse, error) {
	lpr := []ListProfilesResponse{}
	client := &http.Client{}
	resp, err := client.Get(c.ListUrl())
	if err != nil {
		return lpr, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return lpr, err
	}

	if err = json.Unmarshal(body, &lpr); err != nil {
		return lpr, err
	}
	log.Debugf("resp: %s", spew.Sdump(lpr))

	return lpr, nil
}

func (c *ECSClient) Delete(profile string) error {
	req, err := http.NewRequest(http.MethodDelete, c.LoadUrl(profile), bytes.NewBuffer([]byte("")))
	if err != nil {
		return err
	}

	client := &http.Client{}
	req.Header.Set("Content-Type", CHARSET_JSON)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	return CheckDoResponse(resp)
}

// ReadClientRequest unmarshals the client's request into our ClientRequest struct
// used to load new credentials into the server
func ReadClientRequest(r *http.Request) (*ECSClientRequest, error) {
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return &ECSClientRequest{}, fmt.Errorf("reading body: %s", err.Error())
	}
	req := &ECSClientRequest{}
	if err = json.Unmarshal(body, req); err != nil {
		return &ECSClientRequest{}, fmt.Errorf("parsing json: %s", err.Error())
	}
	return req, nil
}

func CheckDoResponse(resp *http.Response) error {
	if resp.StatusCode < 200 || resp.StatusCode > 200 {
		return fmt.Errorf("HTTP Error %d", resp.StatusCode)
	}
	return nil
}
