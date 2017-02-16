// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// cpupower manipulates Linux CPU frequency scaling settings.
package cpupower

import (
	"io/ioutil"
	"path/filepath"
	"regexp"
)

// Domain is a frequency scaling domain. This may include more than
// one CPU.
type Domain struct {
	path     string
	min, max int
}

var policyRe = regexp.MustCompile(`policy\d+$`)

// Domains returns the frequency scaling domains of this host.
func Domains() ([]*Domain, error) {
	dir := "/sys/devices/system/cpu/cpufreq"
	fs, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var domains []*Domain
	for _, f := range fs {
		if !f.IsDir() || !policyRe.MatchString(f.Name()) {
			continue
		}
		pdir := filepath.Join(dir, f.Name())
		min, err := readInt(filepath.Join(pdir, "cpuinfo_min_freq"))
		if err != nil {
			return nil, err
		}
		max, err := readInt(filepath.Join(pdir, "cpuinfo_max_freq"))
		if err != nil {
			return nil, err
		}
		domains = append(domains, &Domain{pdir, min, max})
	}
	return domains, nil
}

// AvailableRange returns the available frequency range this CPU is
// capable of.
func (d *Domain) AvailableRange() (int, int) {
	return d.min, d.max
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
