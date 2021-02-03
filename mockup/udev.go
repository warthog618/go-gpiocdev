// SPDX-License-Identifier: MIT
//
// SPDX-FileCopyrightText: © 2019 Kent Gibson <warthog618@gmail.com>.

package mockup

import (
	"errors"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/pilebones/go-udev/netlink"
)

type udevMonitor struct {
	conn  *netlink.UEventConn
	queue chan netlink.UEvent
	quit  chan struct{}
}

func (m *udevMonitor) Chips(lines []int) ([]Chip, error) {
	evts := make([]netlink.UEvent, len(lines))
	for i := range evts {
		select {
		case evts[i] = <-m.queue:
		case <-time.After(time.Second):
			return nil, errors.New("timeout waiting for udev events")
		}
	}
	sort.Slice(evts, func(i, j int) bool {
		return evts[i].Env["DEVNAME"] < evts[j].Env["DEVNAME"]
	})
	// make chips from udev events
	cc := make([]Chip, len(lines))
	for i, l := range lines {
		devpath := evts[i].Env["DEVNAME"]
		name := devpath[len("/dev/"):]
		var num int
		_, err := fmt.Sscanf(name, "gpiochip%d", &num)
		if err != nil {
			return nil, fmt.Errorf("failed to parse chip num: %s", err)
		}
		cc[i] = Chip{
			Name:      name,
			Label:     fmt.Sprintf("gpio-mockup-%c", 'A'+i),
			Lines:     l,
			DevPath:   devpath,
			DbgfsPath: fmt.Sprintf("/sys/kernel/debug/gpio-mockup/gpiochip%d/", num)}
	}
	return cc, nil
}

func newUdevMonitor() (*udevMonitor, error) {
	conn := new(netlink.UEventConn)
	if err := conn.Connect(netlink.UdevEvent); err != nil {
		return nil, fmt.Errorf("unable to connect to Netlink Kobject UEvent socket")
	}
	action := "add"
	matcher := &netlink.RuleDefinition{Action: &action,
		Env: map[string]string{
			"SUBSYSTEM": "gpio",
			"DEVPATH":   "/devices/platform/gpio-mockup\\.\\d+/gpiochip\\d+",
		}}
	queue := make(chan netlink.UEvent)
	errors := make(chan error)
	quit := conn.Monitor(queue, errors, matcher)
	mon := udevMonitor{conn: conn, queue: queue, quit: quit}
	go func() {
		for {
			select {
			case err := <-errors:
				log.Printf("ERROR: %v", err)
			case <-quit:
				return
			}
		}
	}()
	return &mon, nil
}

func (m *udevMonitor) Close() {
	m.quit <- struct{}{}
	m.conn.Close()
}
