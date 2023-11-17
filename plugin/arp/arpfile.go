// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This file is a modified version of net/hosts.go from the golang repo

// This file is a modified version of plugin/hosts/hostsfile.go from the coredns repo

package arp

import (
	"bufio"
	"bytes"
	"io"
	"net"
	"os"
	"sync"
	"time"
)

// Map contains the name
type Map struct {
	// Key for the list of names must be a MAC address.
	addr map[string][]string
}

func newMap() *Map {
	return &Map{
		addr: make(map[string][]string),
	}
}

// Len returns the total number of addresses in the hostmap.
func (h *Map) Len() int {
	l := 0
	for _, name := range h.addr {
		l += len(name)
	}
	return l
}

// Arpfile contains known ethers entries.
type Arpfile struct {
	sync.RWMutex

	// hosts maps for lookups
	hmap *Map

	// path to the ethers file
	path string

	// mtime and size are only read and modified by a single goroutine
	mtime time.Time
	size  int64
}

// readEthers determines if the cached data needs to be updated based on the size and modification time of the file.
func (h *Arpfile) readEthers() {
	file, err := os.Open(h.path)
	if err != nil {
		// We already log a warning if the file doesn't exist or can't be opened on setup. No need to return the error here.
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return
	}
	h.RLock()
	size := h.size
	h.RUnlock()

	if h.mtime.Equal(stat.ModTime()) && size == stat.Size() {
		return
	}

	newMap, err := h.parse(file)
	if err != nil {
		//TODO
	}
	log.Debugf("Parsed ethers file into %d entries", newMap.Len())

	h.Lock()

	h.hmap = newMap
	// Update the data cache.
	h.mtime = stat.ModTime()
	h.size = stat.Size()

	h.Unlock()
}

// Parse reads the ethers file and populates the map.
func (h *Arpfile) parse(r io.Reader) (*Map, error) {
	hmap := newMap()

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Bytes()
		if i := bytes.Index(line, []byte{'#'}); i >= 0 {
			// Discard comments.
			line = line[0:i]
		}
		f := bytes.Fields(line)
		if len(f) < 2 {
			continue
		}
		addr, err := net.ParseMAC(string(f[0]))
		if err != nil {
			return nil, err
		}
		if addr == nil {
			continue
		}

		for i := 1; i < len(f); i++ {
			name := string(f[i])
			//name = plugin.Name(name).Normalize() //commented to preserve capitalization

			hmap.addr[addr.String()] = append(hmap.addr[addr.String()], name)
		}
	}

	return hmap, nil
}

// LookupStaticAddr looks up the hosts for the given address from the hosts file.
func (h *Arpfile) LookupStaticAddr(addr string) []string {
	hwaddr, err := net.ParseMAC(addr)
	if err != nil {
		return nil //TODO return nil, err
	}
	addrStr := hwaddr.String()
	if addr == "" {
		return nil
	}

	h.RLock()
	defer h.RUnlock()
	hosts1 := h.hmap.addr[addrStr]

	if len(hosts1) == 0 {
		return nil
	}

	return hosts1
}
