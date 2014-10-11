package main

import (
	"dropcam"
	"fmt"
	"time"
)
import "os"

const (
	USER = "DROPCAM_USER"
	PASS = "DROPCAM_PASS"
)

func main() {

	u := os.Getenv(USER)
	p := os.Getenv(PASS)

	if u == "" || p == "" {
		fmt.Printf("need to set both %s and %s\n", USER, PASS)
		return
	}
	fmt.Printf("***** GETTING Dropcam **** \n")
	d, err := new(dropcam.Dropcam).Init(u, p)
	if err != nil {
		fmt.Printf("failed to Init Dropcam Credentials: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("***** GETTING Cameras **** \n")
	c, err := d.Cameras()
	if err != nil {
		fmt.Printf("failed to Get Cameras: %s\n", err)
		os.Exit(1)
	}

	//fmt.Printf("type=%T\n", c.Cam.Items)
	for j, owned := range c.Cam {
		//fmt.Printf("%d %d: %T %T\n", i, j, items, owned)
		fmt.Printf("%d: %s\n", j, owned.Title)
	}

	st := time.Now()
	et := time.Now()
	fmt.Printf("starting at %s ending at %s\n", st, et)

	for {
		fmt.Printf("***** GETTING Image **** \n")
		for i, o := range c.Cam {
			fn := "./rob/img-" + fmt.Sprintf("%d-", i) + fmt.Sprintf("%d", time.Now().Unix())
			err = c.SaveImage(&o, fn, 720, time.Now())
			if err != nil {
				fmt.Printf("error saving image %d\n", i)
			}
			fmt.Printf("saved image %s\n", fn)
		}
		time.Sleep(5 * time.Second)
	}
}
