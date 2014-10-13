// Copyright 2014 Robert Baruch (robertbaruch@mac.com). All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package dropcam implements a basic library to access DropCam cameras.
//
package dropcam

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/lafikl/fluent"
)

// Constants
const (
	NexusBase = "https://nexusapi.dropcam.com"
	ApiBase   = "https://www.dropcam.com"
	ApiPath   = "api/v1"
	Devel     = false
)

// The UserCreds contains the credentials sent to the DropCam URL
type UserCreds struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// The DropCam type lists the URL acess points and contains the credentials and session cookie
type Dropcam struct {
	LoginPath           string
	CamerasGet          string
	CamerasUpdate       string
	CamerasGetVisible   string
	CamerasGetImagePath string
	EventPath           string
	EventGetClipPath    string
	PropertiesPath      string

	Creds  UserCreds
	Cookie string
}

// The Cameras type contains all of the user-owned dropcams associated with the Drocpam object
type Cameras struct {
	Dropcam *Dropcam
	Cam     []Owned
}

// The Events type contains events for a specific camera over a defined epoch
type Events struct {
}

// The Owned type contains the attribuetes associated with a users dropcam
type Owned struct {
	Capabilities        []string    `json:"capabilities"`
	Description         string      `json:"description"`
	DownloadHost        string      `json:"download_host"`
	HasBundle           bool        `json:"has_bundle"`
	HoursOfRecordingMax float64     `json:"hours_of_recording_max"`
	Id                  int64       `json:"id"`
	IsConnected         bool        `json:"is_connected"`
	IsOnline            bool        `json:"is_online"`
	IsPublic            bool        `json:"is_public"`
	IsStreaming         bool        `json:"is_streaming"`
	IsStreamingEnabled  bool        `json:"is_streaming_enabled"`
	IsTrialMode         bool        `json:"is_trial_mode"`
	IsTrialWarning      bool        `json:"is_trial_warning"`
	LastLocalIp         string      `json:"last_local_ip"`
	LiveStreamHost      string      `json:"live_stream_host"`
	Location            interface{} `json:"location"`
	MacAddress          string      `json:"mac_address"`
	Name                string      `json:"name"`
	NestStructureId     interface{} `json:"nest_structure_id"`
	OwnerId             string      `json:"owner_id"`
	PublicToken         string      `json:"public_token"`
	Timezone            string      `json:"timezone"`
	TimezoneUtcOffset   int64       `json:"timezone_utc_offset"`
	Title               string      `json:"title"`
	TrialDaysLeft       int64       `json:"trial_days_left"`
	Type                int64       `json:"type"`
	Uuid                string      `json:"uuid"`
	Where               string      `json:"where"`
}

type Items struct {
	Owned      []Owned       `json:"owned"`
	Subscribed []interface{} `json:"subscribed"`
}

type Cam struct {
	Items []struct {
		Owned      []Owned       `json:"owned"`
		Subscribed []interface{} `json:"subscribed"`
	} `json:"items"`
	Status            int64  `json:"status"`
	StatusDescription string `json:"status_description"`
	StatusDetail      string `json:"status_detail"`
}

// Private Methods
//

func Dbg(format string, args ...interface{}) {
	if !Devel {
		return
	}
	str := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stderr, "%v\n", str)
}

func getBodyRespCode(rb io.ReadCloser) (int, error) {
	body, _ := ioutil.ReadAll(rb)
	// log.Println("response Body:", string(body))

	type BodyStatus struct {
		Status int
	}
	var bStat BodyStatus
	err := json.Unmarshal(body, &bStat)
	if err != nil {
		return 0, err
	}
	// Dbg("response Body code: %d", bStat.Status)
	return bStat.Status, nil
}

func (d *Dropcam) postRequest(url string, uuid string, data interface{}) (resp *http.Response, err error) {

	req := fluent.New()
	req.Post(url).
		InitialInterval(time.Duration(time.Millisecond)).
		Json(data)

	referer := ApiBase + "/" + "watch" + "/" + uuid
	req.SetHeader("Referer", referer)
	req.SetHeader("cookie", d.Cookie)

	resp, err = req.Send()

	log.Println("response Status:", resp.Status)
	log.Println("response Headers:", resp.Header)

	rc, err := getBodyRespCode(resp.Body)
	if err != nil {
		return nil, errors.New("Failed to get Reply Response Code")
	}
	if rc != 200 {
		return nil, errors.New("Malformed Request")
	}

	return resp, nil
}

func (d *Dropcam) getRequest(url string, v url.Values) (resp *http.Response, err error) {

	// Dropcam http request function.

	req := fluent.New()
	if d.Cookie != "" {
		req.SetHeader("cookie", d.Cookie)
	}

	reqUrl := url + "?" + v.Encode()
	Dbg("REQ[%s] =>[%s]\n", d.Cookie, reqUrl)
	req.Get(reqUrl).
		InitialInterval(time.Duration(time.Millisecond)).
		Retry(3)

	resp, err = req.Send()

	if err != nil {
		return nil, err
	}

	return resp, nil
}

// Init is the method that passed the credentials to the dropcam server and receives back a session cookie
// for subsequent requests
func (d *Dropcam) Init(username string, password string) (*Dropcam, error) {

	d.LoginPath = ApiBase + "/" + ApiPath + "/" + "login.login"
	d.CamerasGet = ApiBase + "/" + ApiPath + "/" + "cameras.get"
	d.CamerasUpdate = ApiBase + "/" + ApiPath + "/" + "cameras.update"
	d.CamerasGetVisible = ApiBase + "/" + ApiPath + "/" + "cameras.get_visible"
	d.CamerasGetImagePath = ApiBase + "/" + ApiPath + "/" + "cameras.get_image"
	d.EventPath = NexusBase + "/" + "get_cuepoint"
	d.EventGetClipPath = NexusBase + "/" + "get_event_clip"
	d.PropertiesPath = ApiBase + "/" + "app/cameras/properties"
	// Creates a new dropcam API instance.

	d.Creds.Username = username
	d.Creds.Password = password
	d.Cookie = ""

	err := d.login()
	if err != nil {
		return nil, err
	}

	return d, nil
}

func (d *Dropcam) login() error {

	v := url.Values{}
	v.Set("username", d.Creds.Username)
	v.Add("password", d.Creds.Password)

	response, err := d.getRequest(d.LoginPath, v)
	if err != nil {
		errStr := fmt.Sprintf("Login Request Failed: %s", err)
		return errors.New(errStr)
	}
	d.Cookie = response.Header.Get("Set-Cookie")
	if d.Cookie == "" {
		errStr := fmt.Sprintf("Login Returned No Cookie")
		return errors.New(errStr)
	}
	Dbg("setting cookie -> [%s]\n", d.Cookie)
	return nil

}

// The Cameras method will return a list of DropCam cameras from the server.
// These are soley private cameras owned by the credentials.
func (d *Dropcam) Cameras() (*Cameras, error) {
	// returns: list of Camera class objects

	if d.Cookie == "" {
		return nil, d.login()
	}

	v := url.Values{}
	v.Set("group_cameras", "True")

	response, err := d.getRequest(d.CamerasGetVisible, v)
	if err != nil {
		return nil, errors.New("Get Visible Cameras Request Failed")
	}

	body, err := ioutil.ReadAll(response.Body)

	var cam Cam
	err = json.Unmarshal(body, &cam)
	if err != nil {
		fmt.Println("error:", err)
		return nil, err
	}

	cameras := new(Cameras)
	cameras.Dropcam = d

	for _, items := range cam.Items {
		for _, owned := range items.Owned {
			//fmt.Printf("%d: %s\n", j, owned)
			cameras.Cam = append(cameras.Cam, owned)
		}
	}
	return cameras, nil
}

type CamProp struct {
	UUID  string `json:"camera_uuid"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

// The SetProperties method will set varias properties on an individual
// Owned Camera
func (c *Cameras) SetProperties(o *Owned, name string, value string) (bool, error) {

	// Changes a property on the camera
	// Examples:
	// irled.state: auto_on / always_on / always_off
	// streaming.enabled: true / false
	// streaming.params.hd: true / false
	// audio.enabled: true / false
	// statusled.enabled: true / false

	url := c.Dropcam.PropertiesPath + o.Uuid

	props := new(CamProp)
	props.UUID = o.Uuid
	props.Name = name
	props.Value = value

	resp, err := c.Dropcam.postRequest(url, o.Uuid, props)
	if err != nil {
		return false, errors.New("Failed postRequest ")
	}

	rc, err := getBodyRespCode(resp.Body)
	if err != nil {
		return false, errors.New("Failed to get Reply Response Code")
	}
	if rc != 200 {
		return false, errors.New("SeProperties Malformed Request")
	}

	return true, nil
}

// The GetEvents method will return an array of Events for the given timeframe
func (c *Cameras) GetEvents(o *Owned, st time.Time, et time.Time) ([]Events, error) {
	// Returns a list of camera events for a given time period:

	//:param start: start time in seconds since epoch
	//:param end: end time in seconds since epoch (defaults to current time)
	//:returns: list of Event class objects

	//events := new([]Events)

	fmt.Printf("STARTING AT: [%s]\n", st)
	v := url.Values{}
	v.Set("uuid", o.Uuid)
	v.Add("start_time", fmt.Sprintf("%d", st.Unix()))
	v.Add("end_time", fmt.Sprintf("%d", et.Unix()-60*60*24))
	v.Add("human", "True")

	response, err := c.Dropcam.getRequest(c.Dropcam.EventPath, v)
	if err != nil {
		Dbg("Events request failed\n")
		return nil, errors.New("Get Visible Cameras Request Failed")
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		Dbg("Failed to Read Event Body\n")
		return nil, errors.New("EVent ioutil.ReadAll failed")
	}
	Dbg("Camera Response body = [%s]\n", body)

	var event Events
	err = json.Unmarshal(body, &event)
	if err != nil {
		fmt.Println("Can't unmarshall Events", err)
		return nil, err
	}

	//events.append(Event(self, item))
	return nil, nil
}

func (c *Cameras) getImage(o *Owned, width int, st time.Time) ([]byte, error) {

	// Requests a camera image, returns response object.

	v := url.Values{}
	v.Set("uuid", o.Uuid)
	v.Add("width", fmt.Sprintf("%d", width))

	/*
		if st != "" {
			v.Add("time", fmt.Sprintf("%d", st.Unix()))
		}
	*/

	response, err := c.Dropcam.getRequest(c.Dropcam.CamerasGetImagePath, v)
	if err != nil {
		return nil, errors.New("Get Image Failed")
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, errors.New("Failed to get Reply Response Code")
	}
	if response.StatusCode != 200 || (response.Header.Get("content-length") == "0") {
		return nil, errors.New("Malformed Request or image has 0 size")
	}

	return body, nil
}

// The SaveImage method retrieves an image from a specifically Owned camera
// and writes it to disk.
func (c *Cameras) SaveImage(o *Owned, path string, width int, st time.Time) error {
	// Saves a camera image to disc.

	Dbg("***** getting image *****\n")
	img, err := c.getImage(o, width, st)
	if err != nil {
		Dbg("Failed to getImage: %s\n", err)
		return err
	}

	err = ioutil.WriteFile(path, img, 0644)
	if err != nil {
		Dbg("failed to write image into file: '%s', err\n", path, err)
		return err
	}

	Dbg("wrote image to \"%s\"\n", path)
	return nil

}
