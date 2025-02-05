// PTRA: Patient Trajectory Analysis Library
// Copyright (c) 2022 imec vzw.

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version, and Additional Terms
// (see below).

// This program is distributed in the hope that it will be useful, but
// WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
// Affero General Public License for more details.

// You should have received a copy of the GNU Affero General Public
// License and Additional Terms along with this program. If not, see
// <https://github.com/ExaScience/ptra/blob/master/LICENSE.txt>.

package ptra

import (
	"slices"
	"strings"
)

// PatientFilter prescribes a function type for implementing filters on TriNetX patients, to be able to calculate
// trajectories for specific cohorts. E.g. male patients, patients <70 years, patients with specific cancer stage, etc.
type PatientFilter func(patient *Patient) bool

// TrajectoryFilter is a type to define a trajectory filter function. Such filters take as input a trajectory and must
// return a bool as output that determines if a trajectory passes a filter or not.
type TrajectoryFilter func(t *Trajectory) bool

func GetPatientFilter(s string, tinfo map[string][]*TumorInfo) PatientFilter {
	id := func(p *Patient) bool { return true }
	switch s {
	case "id":
		return id
	case "age70+":
		return AboveSeventyAggregator()
	case "age70-":
		return LessThanSeventyAggregator()
	case "male":
		return FemaleFilter()
	case "female":
		return MaleFilter()
	case "Ta":
		return TaStageAggregator(tinfo)
	case "T1":
		return T1StageAggregator(tinfo)
	case "Tis":
		return TisStageAggregator(tinfo)
	case "T2":
		return T2StageAggregator(tinfo)
	case "T3":
		return T3StageAggregator(tinfo)
	case "T4":
		return T4StageAggregator(tinfo)
	case "N0":
		return N0StageAggregator(tinfo)
	case "N1":
		return N1StageAggregator(tinfo)
	case "N2":
		return N2StageAggregator(tinfo)
	case "N3":
		return N3StageAggregator(tinfo)
	case "M0":
		return M0StageAggregator(tinfo)
	case "M1":
		return M1StageAggregator(tinfo)
	case "EOI-":
		return EOIAfterFilter()
	case "EOI+":
		return EOIBeforeFilter()
	case "MIBC":
		return MIBCAggregator(tinfo)
	case "NMIBC":
		return NMIBCAggregator(tinfo)
	case "mUC":
		return MUCAggregator(tinfo)
	default:
		return id
	}
}

func GetPatientFilters(filters string, tinfo map[string][]*TumorInfo) []PatientFilter {
	var result []PatientFilter
	for _, f := range strings.Split(filters, ",") {
		trimmed := strings.Trim(f, " ")
		result = append(result, GetPatientFilter(trimmed, tinfo))
	}
	return result
}

func GetTrajectoryFilters(filters string, exp *Experiment) []TrajectoryFilter {
	var result []TrajectoryFilter
	for _, f := range strings.Split(filters, ",") {
		trimmed := strings.Trim(f, " ")
		result = append(result, GetTrajectoryFilter(trimmed, exp))
	}
	return result
}

func GetTrajectoryFilter(filter string, exp *Experiment) TrajectoryFilter {
	id := func(t *Trajectory) bool { return true }
	switch filter {
	case "neoplasm":
		return CancerTrajectoryFilter(exp)
	case "bc":
		return BladderCancerTrajectoryFilter(exp)
	default:
		return id
	}
}

// cancerStageAggregator filters a set of patients to only include those that satisfy a given predicate that is applied
// on the patient's tumor information (which encodes cancer stages etc).
func cancerStageAggregator(predicate func(tInfo *TumorInfo) bool, tInfoMap map[string][]*TumorInfo) PatientFilter {
	return func(p *Patient) bool {
		//multiple tumor info entries per patient possible
		if tInfos, ok := tInfoMap[p.PIDString]; ok {
			if !ok {
				return false
			}
			tInfoToUseIndex := -1
			for i, tInfo := range tInfos { //go over all infos to grab the latest cancer stage that satisfies the predicate
				if predicate(tInfo) {
					tInfoToUseIndex = i
				}
			}
			if tInfoToUseIndex != -1 {
				// have a patient with specific cancer stage diagnosis
				// filter out diagnoses at later dates if possibly followed by other cancer stage
				if tInfoToUseIndex+1 < len(tInfos) {
					nextStageDate := tInfos[tInfoToUseIndex+1].Date
					newD := []*Diagnosis{}
					for _, d := range p.Diagnoses {
						if DiagnosisDateSmallerThan(d.Date, nextStageDate) {
							newD = append(newD, d)
						} else {
							continue
						}
					}
					p.Diagnoses = newD
				}
				return true
			}
		}
		return false
	}
}

// NMIBCAggregator checks all patients if they match the cancer criteria to be defined as non muscle invasive bladder
// cancer patients.
func NMIBCAggregator(tinfoMap map[string][]*TumorInfo) PatientFilter {
	return cancerStageAggregator(func(tInfo *TumorInfo) bool {
		if tInfo.TStage == "Tis" || tInfo.TStage == "Ta" ||
			(tInfo.TStage == "T1" && tInfo.NStage == "N0" && tInfo.MStage == "M0") {
			return true
		}
		return false
	}, tinfoMap)
}

// MIBCAggregator checks all patients if they match the cancer criteria to be defined as muscle invasive bladder cancer
// patients.
func MIBCAggregator(tinfoMap map[string][]*TumorInfo) PatientFilter {
	return cancerStageAggregator(func(tInfo *TumorInfo) bool {
		if tInfo.TStage == "T2" || tInfo.TStage == "T3" ||
			(tInfo.TStage == "T4" && tInfo.MStage == "M0" &&
				(tInfo.NStage == "N0" || tInfo.NStage == "N1" || tInfo.NStage == "N2" || tInfo.NStage == "N3")) {
			return true
		}
		return false
	}, tinfoMap)
}

// MUCAggregator checks all patients if they match the cancer criteria to be defined as metastisized bladder cancer
// patients.
func MUCAggregator(tinfoMap map[string][]*TumorInfo) PatientFilter {
	return cancerStageAggregator(func(tInfo *TumorInfo) bool {
		if tInfo.MStage == "M0" {
			return true
		}
		return false
	}, tinfoMap)
}

// TaStageAggregator collects patients with stage Ta bladder cancer.
func TaStageAggregator(tinfoMap map[string][]*TumorInfo) PatientFilter {
	return cancerStageAggregator(func(tInfo *TumorInfo) bool {
		if tInfo.TStage == "Ta" {
			return true
		}
		return false
	}, tinfoMap)
}

// T1StageAggregator collects patients with stage T1 bladder cancer.
func T1StageAggregator(tinfoMap map[string][]*TumorInfo) PatientFilter {
	return cancerStageAggregator(func(tInfo *TumorInfo) bool {
		if tInfo.TStage == "T1" || tInfo.TStage == "T1a" || tInfo.TStage == "T1c" {
			return true
		}
		return false
	}, tinfoMap)
}

// TisStageAggregator collects patients with stage Tis bladder cancer.
func TisStageAggregator(tinfoMap map[string][]*TumorInfo) PatientFilter {
	return cancerStageAggregator(func(tInfo *TumorInfo) bool {
		if tInfo.TStage == "Tis" {
			return true
		}
		return false
	}, tinfoMap)
}

// T2StageAggregator collects patients with stage T2 bladder cancer.
func T2StageAggregator(tinfoMap map[string][]*TumorInfo) PatientFilter {
	return cancerStageAggregator(func(tInfo *TumorInfo) bool {
		if tInfo.TStage == "T2" || tInfo.TStage == "T2a" || tInfo.TStage == "T2b" || tInfo.TStage == "T2c" {
			return true
		}
		return false
	}, tinfoMap)
}

// T3StageAggregator collects patients with stage T3 bladder cancer.
func T3StageAggregator(tinfoMap map[string][]*TumorInfo) PatientFilter {
	return cancerStageAggregator(func(tInfo *TumorInfo) bool {
		if tInfo.TStage == "T3" || tInfo.TStage == "T3a" || tInfo.TStage == "T3b" {
			return true
		}
		return false
	}, tinfoMap)
}

// T4StageAggregator collects patients with stage T4 bladder cancer.
func T4StageAggregator(tinfoMap map[string][]*TumorInfo) PatientFilter {
	return cancerStageAggregator(func(tInfo *TumorInfo) bool {
		if tInfo.TStage == "T4" || tInfo.TStage == "T4a" || tInfo.TStage == "T4b" {
			return true
		}
		return false
	}, tinfoMap)
}

// N0StageAggregator collects patients with stage N0 bladder cancer.
func N0StageAggregator(tinfoMap map[string][]*TumorInfo) PatientFilter {
	return cancerStageAggregator(func(tInfo *TumorInfo) bool {
		if tInfo.NStage == "N0" {
			return true
		}
		return false
	}, tinfoMap)
}

// N1StageAggregator collects patients with stage N1 bladder cancer.
func N1StageAggregator(tinfoMap map[string][]*TumorInfo) PatientFilter {
	return cancerStageAggregator(func(tInfo *TumorInfo) bool {
		if tInfo.NStage == "N1" {
			return true
		}
		return false
	}, tinfoMap)
}

// N2StageAggregator collects patients with stage N2 bladder cancer.
func N2StageAggregator(tinfoMap map[string][]*TumorInfo) PatientFilter {
	return cancerStageAggregator(func(tInfo *TumorInfo) bool {
		if tInfo.NStage == "N2" {
			return true
		}
		return false
	}, tinfoMap)
}

// N3StageAggregator collects patients with stage N0 bladder cancer.
func N3StageAggregator(tinfoMap map[string][]*TumorInfo) PatientFilter {
	return cancerStageAggregator(func(tInfo *TumorInfo) bool {
		if tInfo.NStage == "N3" {
			return true
		}
		return false
	}, tinfoMap)
}

// M0StageAggregator collects patients with stage M0 bladder cancer.
func M0StageAggregator(tinfoMap map[string][]*TumorInfo) PatientFilter {
	return cancerStageAggregator(func(tInfo *TumorInfo) bool {
		if tInfo.MStage == "M0" {
			return true
		}
		return false
	}, tinfoMap)
}

// M1StageAggregator collects patients with stage M1 bladder cancer.
func M1StageAggregator(tinfoMap map[string][]*TumorInfo) PatientFilter {
	return cancerStageAggregator(func(tInfo *TumorInfo) bool {
		if tInfo.MStage == "M1" || tInfo.MStage == "M1a" || tInfo.MStage == "M1b" {
			return true
		}
		return false
	}, tinfoMap)
}

// CancerTrajectoryFilter filters trajectories down to trajectories that contain at least one diagnosis code that is
// considered to be cancer-related, e.g. containing the word "neoplasm".
func CancerTrajectoryFilter(exp *Experiment) TrajectoryFilter {
	//Determine all diagnosis codes that are cancer-related
	CancerRelatedMap := map[int]bool{}
	for did, icd10 := range exp.Icd10Map {
		medWords := strings.Split(icd10.Name, " ")
		cancerRelated := false
		for _, word := range medWords {
			if word == "neoplasm" || word == "Neoplasm" {
				cancerRelated = true
				break
			}
		}
		CancerRelatedMap[did] = cancerRelated
	}
	return func(t *Trajectory) bool {
		for _, did := range t.Diagnoses {
			if CancerRelatedMap[did] {
				return true
			}
		}
		return false
	}
}

// BladderCancerTrajectoryFilter filters trajectories down to trajectories with
// at least one diagnosis that is related to bladder cancer specifically,
// cf. ICD10 Categories C67,C77,C78,C79 or procedures such as MVAC chemo,
// IVT treatment, or radical cystectomy.
func BladderCancerTrajectoryFilter(exp *Experiment) TrajectoryFilter {
	// Determine all internal diagnosis codes that are bladder cancer-related
	bladderCancerCodes := []string{
		"C67", "C77", "C78",
		// also check self-defined codes for treatments
		"C79", "C98", "C99", "C100",
	}

	bladderCancerRelatedMap := map[int]bool{}
	for did, icdCode := range exp.IdMap {
		if len(icdCode) >= 3 {
			subCode := icdCode[0:3]
			bladderCancerRelatedMap[did] = slices.Contains(bladderCancerCodes, subCode) || (len(icdCode) >= 4 && icdCode[0:4] == "C100")
		}
	}
	return func(t *Trajectory) bool {
		for _, did := range t.Diagnoses {
			if bladderCancerRelatedMap[did] {
				return true
			}
		}
		return false
	}
}

func ApplyPatientFilter(filter PatientFilter, pMap *PatientMap) *PatientMap {
	newPMap := &PatientMap{PIDStringMap: map[string]int{}, PIDMap: map[int]*Patient{}, Ctr: pMap.Ctr}
	for pid, p := range pMap.PIDMap {
		if filter(p) {
			newPMap.PIDStringMap[p.PIDString] = pid
			newPMap.PIDMap[pid] = p
			if p.Sex == Male {
				newPMap.MaleCtr++
			} else {
				newPMap.FemaleCtr++
			}
		}
	}
	return newPMap
}

func ApplyPatientFilters(filters []PatientFilter, pMap *PatientMap) *PatientMap {
	newPMap := &PatientMap{PIDStringMap: map[string]int{}, PIDMap: map[int]*Patient{}, Ctr: pMap.Ctr}
	for pid, p := range pMap.PIDMap {
		res := true
		for _, filter := range filters {
			res = filter(p) && res
			if !res {
				break
			}
		}
		if res {
			newPMap.PIDStringMap[p.PIDString] = pid
			newPMap.PIDMap[pid] = p
			if p.Sex == Male {
				newPMap.MaleCtr++
			} else {
				newPMap.FemaleCtr++
			}
		}
	}
	return newPMap
}

// SexFilter removes all patients of the given sex.
func SexFilter(sex int) PatientFilter {
	return func(p *Patient) bool {
		return p.Sex != sex
	}
}

// MaleFilter removes all male patients.
func MaleFilter() PatientFilter {
	return SexFilter(Male)
}

// FemaleFilter removes all female patients.
func FemaleFilter() PatientFilter {
	return SexFilter(Female)
}

// EOIFilter removes all diagnoses for patients that satisfy a given predicate
func EOIFilter(test func(d1, d2 DiagnosisDate) bool) PatientFilter {
	return func(p *Patient) bool {
		if p.EOIDate == nil { //skip patients without EOIDate
			return false
		}
		newD := []*Diagnosis{}
		for _, d := range p.Diagnoses {
			if test(d.Date, *p.EOIDate) {
				break
			}
			newD = append(newD, d)
		}
		p.Diagnoses = newD
		if newD == nil {
			return false
		}
		return true
	}
}

// EOIBeforeFilter removes all diagnoses before the event of interest date
func EOIBeforeFilter() PatientFilter {
	return EOIFilter(func(d1, d2 DiagnosisDate) bool { return DiagnosisDateSmallerThan(d1, d2) })
}

// EOIAfterFilter removes all diagnoses after the event of interest date
func EOIAfterFilter() PatientFilter {
	return EOIFilter(func(d1, d2 DiagnosisDate) bool { return DiagnosisDateSmallerThan(d2, d1) })
}

// ageLessAggregator collects all patients younger than a specific age or trims down their data up until that age.
func ageLessAggregator(age int) PatientFilter {
	return func(p *Patient) bool {
		fYear := p.YOB + age - 1 // last year with diagnosis accepted
		//remove all diagnoses past a specific age
		newD := []*Diagnosis{}
		for _, d := range p.Diagnoses {
			if d.Date.Year > fYear {
				break
			}
			newD = append(newD, d)
		}
		p.Diagnoses = newD
		if len(newD) == 0 {
			return false
		}
		return true
	}
}

// ageAboveAggretator collects all patients older than a specific age and removes all diagnoses before that date.
func ageAboveAggregator(age int) PatientFilter {
	return func(p *Patient) bool {
		mYear := p.YOB + age // min year with diagnosis accepted
		//remove all diagnoses before a specific age
		newD := []*Diagnosis{}
		for _, d := range p.Diagnoses {
			if d.Date.Year <= mYear {
				continue
			}
			newD = append(newD, d)
		}
		p.Diagnoses = newD
		if len(newD) == 0 {
			return false
		}
		return true
	}
}

// LessThanSeventyAggregator collects all patients below a specific age.
func LessThanSeventyAggregator() PatientFilter {
	return ageLessAggregator(70)
}

// AboveSeventyAggregator collects all patients above a specific age.
func AboveSeventyAggregator() PatientFilter {
	return ageAboveAggregator(70)
}
