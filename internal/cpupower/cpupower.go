// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// cpupower manipulates Linux CPU frequency scaling settings.
package cpupower

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
)

// Domain is a frequency scaling domain. This may include more than
// one CPU.
type Domain struct {
	path      string
	min, max  int
	available []int
}

var cpuRe = regexp.MustCompile(`cpu\d+$`)

// Domains returns the frequency scaling domains of this host.
func Domains() ([]*Domain, error) {
	dir := "/sys/devices/system/cpu"
	fs, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var domains []*Domain
	haveDomains := make(map[string]bool)
	for _, f := range fs {
		if !f.IsDir() || !cpuRe.MatchString(f.Name()) {
			continue
		}
		pdir := filepath.Join(dir, f.Name(), "cpufreq")

		// Get the frequency domain, if any.
		cpus, err := ioutil.ReadFile(filepath.Join(pdir, "freqdomain_cpus"))
		if err == nil {
			if haveDomains[string(cpus)] {
				// We already have a CPU in this domain.
				continue
			}
			haveDomains[string(cpus)] = true
		} else if !os.IsNotExist(err) {
			return nil, err
		}

		min, err := readInt(filepath.Join(pdir, "cpuinfo_min_freq"))
		if err != nil {
			return nil, err
		}
		max, err := readInt(filepath.Join(pdir, "cpuinfo_max_freq"))
		if err != nil {
			return nil, err
		}
		avail, err := readInts(filepath.Join(pdir, "scaling_available_frequencies"))
		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}
		sort.Ints(avail)
		domains = append(domains, &Domain{pdir, min, max, avail})
	}
	return domains, nil
}

// AvailableRange returns the available frequency range this CPU is
// capable of and the set of available frequencies in ascending order
// or nil if any frequency can be set.
func (d *Domain) AvailableRange() (int, int, []int) {
	return d.min, d.max, d.available
}

// CurrentRange returns the current frequency range this CPU's
// governor can select between.
func (d *Domain) CurrentRange() (int, int, error) {
	min, err := readInt(filepath.Join(d.path, "scaling_min_freq"))
	if err != nil {
		return 0, 0, err
	}
	max, err := readInt(filepath.Join(d.path, "scaling_max_freq"))
	if err != nil {
		return 0, 0, err
	}
	return min, max, nil
}

// SetRange sets the frequency range this CPU's governor can select
// between.
func (d *Domain) SetRange(min, max int) error {
	// Attempting to set an empty range will cause an IO error.
	// Rather than trying to figure out the right order to set
	// them in, try both orders.
	err1 := writeInt(filepath.Join(d.path, "scaling_min_freq"), min)
	if err2 := writeInt(filepath.Join(d.path, "scaling_max_freq"), max); err2 != nil {
		return err2
	}
	if err1 != nil {
		err1 = writeInt(filepath.Join(d.path, "scaling_min_freq"), min)
	}
	return err1
}
