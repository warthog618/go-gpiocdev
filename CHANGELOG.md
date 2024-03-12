<!--
SPDX-FileCopyrightText: 2024 Kent Gibson <warthog618@gmail.com>

SPDX-License-Identifier: MIT
-->
## [Unreleased](https://github.com/warthog618/gpiod/compare/v0.8.3...HEAD)

## v0.8.3 - 2024-03-16

- deprecate in favour of **go-gpiocdev**

## v0.8.2 - 2023-07-28

- switch tests from **gpio-mockup** to **gpio-sim**.
- drop test dependency on *pilebones/go-udev*.
- drop example dependency on *warthog618/config*.

## v0.8.1 - 2022-12-31

- add bananapi pin mappings.
- fix config check in **gpioset**.

## v0.8.0 - 2022-02-13

- add top level *RequestLine* and *RequestLines* functions to simplify common use cases.
- **blinker** and **watcher** examples interwork with each other on a Raspberry Pi with a jumper across **J8-15** and **J8-16**.
- fix deadlock in **gpiodctl set** no-wait.

## v0.7.1 - 2021-10-10

- restore LICENSE file for go.dev.

## v0.7.0 - 2021-10-08

- *LineEvent* exposes sequence numbers for uAPI v2 events.
- Info tools (**gpiodctl info** and **gpioinfo**) report debounce-period.
- **gpiodctl mon** and watcher example report event sequence numbers.
- **gpiodctl mon** supports setting debounce period.
- **gpiodctl detect** reports kernel uAPI version in use.
- Watchers use Eventfd instead of pipes to reduce open file descriptors.
- start migrating to Go 1.17 go:build style build tags.
- make licensing [REUSE](https://reuse.software/) compliant.

## v0.6.0 - 2020-12-12

- *gpiod* now supports both the old GPIO uAPI (v1) and the newer (v2) introduced
  in Linux 5.10. The library automatically detects the available uAPI versions
  and makes use of the latest.
- applications written for uAPI v1 will continue to work with uAPI v2.
- applications that make use of v2 specific features will return errors when run
  on Linux kernels prior to 5.10.

Breaking API changes:

1. The event handler parameter has been moved from edge options into the
   *WithEventHandler(eh)* option to allow for reconfiguration of edge detection
   which is supported in Linux 5.10.

   Old edge options should be replaced with the *WithEventHandler* option and
   the now parameterless edge option, e.g.:

   ```sed
   s/gpiod\.WithBothEdges(/gpiod.WithBothEdges, gpiod.WithEventHandler(/g
   ```

2. *WithBiasDisable* is renamed *WithBiasDisabled*.  This option is probably
   rarely used and the renaming is trivial, so no backward compatibility is
   provided.

3. *FindLine* has been dropped as line names are not guaranteed to be unique.
   Iterating over the available chips and lines to search for line by name can
   be easily done - the *Chips* function provides the list of available chips as
   a starting point.

   Refer to the *find* command in **gpiodctl** for example code.
