package mockup

import (
	"fmt"

	"github.com/pilebones/go-udev/netlink"
)

type udevMonitor struct {
	conn  *netlink.UEventConn
	queue chan netlink.UEvent
	quit  chan struct{}
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
			<-errors
			//err := <-errors
			//log.Printf("ERROR: %v", err)
		}
	}()
	return &mon, nil
}

func (m *udevMonitor) close() {
	m.quit <- struct{}{}
	m.conn.Close()
}

func (m *udevMonitor) waitEvents(evts []netlink.UEvent) {
	for i := range evts {
		evts[i] = <-m.queue
	}
}
