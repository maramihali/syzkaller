// Copyright 2021 syzkaller project authors. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.

package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/syzkaller/pkg/mgrconfig"
	"github.com/google/syzkaller/pkg/osutil"
	"github.com/google/syzkaller/prog"
	"github.com/google/syzkaller/vm"

	"github.com/google/syzkaller/sys/targets"
)

type cfgFlagVals []string
func (cfvs *cfgFlagVals) String() string {
	return fmt.Sprint(*cfvs)
}

func (cfvs *cfgFlagVals) Set(value string) error {
	if len(*cfvs) > 0 {
		return errors.New("configs flag were already set")
	}
	for _, cfv := range strings.Split(value, ",") {
		*cfvs = append(*cfvs, cfv)
	}
	return nil
}

var (
	cfvs cfgFlagVals
	flagDebug = flag.Bool("debug", false, "dump all VM output to console")
	workdir string
)

// Verifier TODO
type Verifier struct {
	cfgs []*mgrconfig.Config
	vmPools []*vm.Pool
	targets []*prog.Target
	sysTargets []*targets.Target
	crashdir string

	// TODO: decide whether to add a report.Reporter for each vm.Pool or implement something difference

	// TODO: consider connecting to the dashboard
}

func main() {
	flag.Var(&cfvs, "configs", "list of comma-sepatated configuration files")
	flag.Parse()

	cfgs := make([]*mgrconfig.Config, len(cfvs))
	for idx, cfv := range cfvs {
		var err error
		cfgs[idx], err = mgrconfig.LoadFile(cfv)
		if err != nil {
			log.Fatalf("%v", err)
		}
	}

	vmPools := make([]*vm.Pool, len(cfgs))
	for idx, cfg := range cfgs {
		var err error
		vmPools[idx], err = vm.Create(cfg, *flagDebug)
		if err != nil {
			log.Fatalf("%v", err)
		} 
	}

	workdir = cfgs[0].Workdir
	for idx := 1; idx < len(cfgs); idx ++ {
		if workdir != cfgs[idx].Workdir {
			log.Fatalf("working directory mismatch")
		}
	}

	sysTargets := make([]*targets.Target, len(cfgs))
	for idx, cfg := range cfgs {
		sysTargets[idx] = cfg.SysTarget
	}
	targets := make([]*prog.Target, len(cfgs))
	for idx, cfg := range cfgs {
		targets[idx] = cfg.Target
	}

	defer cleanup()
	crashdir := filepath.Join(workdir, "crashes")
	osutil.MkdirAll(crashdir)
	for idx, target := range targets {
		// TODO: this is a hack for comparing the same kernel against itself, remove it afterwards
		targetPath := target.OS + "-" + target.Arch + "-" + strconv.Itoa(idx)
		osutil.Mkdir(filepath.Join(workdir,targetPath))
		osutil.Mkdir(filepath.Join(crashdir, targetPath))
	}

	vrf := &Verifier{
		cfgs: cfgs,
		vmPools: vmPools,
		targets: targets,
		sysTargets: sysTargets,
		crashdir: crashdir,
	}

	log.Printf("Verifier initialised: %v", vrf)
 }

 func cleanup() {
	osutil.RemoveAll(workdir)
 }