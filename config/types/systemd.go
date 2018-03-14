// Copyright 2016 CoreOS, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package types

import (
	"fmt"
	"strings"

	ignTypes "github.com/coreos/ignition/config/v2_1/types"
	"github.com/coreos/ignition/config/validate/astnode"
	"github.com/coreos/ignition/config/validate/report"

	sdunit "github.com/coreos/go-systemd/unit"
)

type Systemd struct {
	Units []SystemdUnit `yaml:"units"`
}

type SystemdUnit struct {
	Name     string              `yaml:"name"`
	Enable   bool                `yaml:"enable"`
	Enabled  *bool               `yaml:"enabled"`
	Mask     bool                `yaml:"mask"`
	Contents string              `yaml:"contents"`
	Dropins  []SystemdUnitDropIn `yaml:"dropins"`
}

type SystemdUnitDropIn struct {
	Name     string `yaml:"name"`
	Contents string `yaml:"contents"`
}

func init() {
	register(func(in Config, ast astnode.AstNode, out ignTypes.Config, platform string) (ignTypes.Config, report.Report, astnode.AstNode) {
		var rep report.Report
		for _, unit := range in.Systemd.Units {
			newUnit := ignTypes.Unit{
				Name:     unit.Name,
				Enable:   unit.Enable,
				Enabled:  unit.Enabled,
				Mask:     unit.Mask,
				Contents: unit.Contents,
			}

			for _, dropIn := range unit.Dropins {
				newUnit.Dropins = append(newUnit.Dropins, ignTypes.Dropin{
					Name:     dropIn.Name,
					Contents: dropIn.Contents,
				})
			}

			unitRep := validateUnit(newUnit)
			rep.Merge(unitRep)
			out.Systemd.Units = append(out.Systemd.Units, newUnit)
		}
		return out, rep, ast
	})
}

func validateUnit(unit ignTypes.Unit) report.Report {
	var rep report.Report

	isEnabled := unit.Enable || (unit.Enabled != nil && *unit.Enabled)

	if unit.Contents != "" {
		parsedUnit, err := sdunit.Deserialize(strings.NewReader(unit.Contents))
		if err != nil {
			rep.Add(report.Entry{
				Kind:    report.EntryError,
				Message: fmt.Sprintf("systemd unit %q could not be parsed: %v", unit.Name, err),
			})
			return rep
		}

		noInstall := true
		for _, section := range parsedUnit {
			if section.Section == "Install" {
				noInstall = false
			}
		}

		if isEnabled && noInstall {
			rep.Add(report.Entry{
				Kind:    report.EntryWarning,
				Message: fmt.Sprintf("systemd unit %q has no [Install] section; 'enabled' will do nothing", unit.Name),
			})
		}
	}

	return rep
}
