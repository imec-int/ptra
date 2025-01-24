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

package ptra_test

import (
	"fmt"
	"github.com/imec-int/ptra/lib"
	"testing"
)

func TestParseIcd10XML(t *testing.T) {
	file := "./icd10cm_tabular_2022.xml"
	icd10XML := lib.ParseIcd10HierarchyFromXml(file)
	lib.PrintIcd10Hierarchy(icd10XML)
}

func TestInitializeIcd10NameMap(t *testing.T) {
	file := "./icd10cm_tabular_2022.xml"
	icd10Names := lib.InitializeIcd10NameMap(file)
	lib.PrintIcd10NameMap(icd10Names)
}

func TestInitializeICD10AnalysisMap(t *testing.T) {
	file := "./icd10cm_tabular_2022.xml"
	icd10Names := lib.InitializeIcd10NameMap(file)
	lib.InitializeIcd10AnalysisMaps(icd10Names, 0)
	lib.InitializeIcd10AnalysisMaps(icd10Names, 1)
	lib.InitializeIcd10AnalysisMaps(icd10Names, 2)
	lib.InitializeIcd10AnalysisMaps(icd10Names, 3)
	lib.InitializeIcd10AnalysisMaps(icd10Names, 4)
	lib.InitializeIcd10AnalysisMaps(icd10Names, 5)
	lib.InitializeIcd10AnalysisMaps(icd10Names, 6)
}

func TestParseTrinetXPatients(t *testing.T) {
	file := "./patient.csv"
	nofCohortAges := 10
	lib.ParseTriNetXPatientData(file, nofCohortAges)
}

func TestInitializeCohorts(t *testing.T) {
	file1 := "./patient.csv"
	nofCohortAges := 10
	patients, _ := lib.ParseTriNetXPatientData(file1, nofCohortAges)
	file2 := "./diagnosis.csv"
	file3 := "./icd10cm_tabular_2022.xml"
	level := 0
	analysisMaps := lib.InitializeIcd10AnalysisMapsFromXML(file3, level)
	lib.ParseTrinetXPatientDiagnoses(file2, "", patients, analysisMaps, map[string]string{})
	nofDiagnosisCodes := analysisMaps.NofDiagnosisCodes
	nofRegions := 1
	cohorts := lib.InitCohorts(patients, nofCohortAges, nofRegions, nofDiagnosisCodes)
	for _, cohort := range cohorts {
		cohort.Log(18)
	}
	fmt.Println("Map DID -> Medical Name")
	collected := make([][]string, len(analysisMaps.Icd10Map))
	for k, v := range analysisMaps.Icd10Map {
		collected[k] = append(collected[k], v.Name)
	}
	for _, v := range collected {
		fmt.Println(v)
		if len(v) > 1 {
			fmt.Println("foo")
		}
	}
	fmt.Println("Map Medical Name -> DID")
	collected2 := make([][]string, len(analysisMaps.Icd10Map))
	for k, v := range analysisMaps.DIDMap {
		collected2[v] = append(collected2[v], k)
	}
	for i, v := range collected2 {
		fmt.Println(collected[i], " : ", v)
	}
}

func TestParseTrinetXPatientDiagnoses(t *testing.T) {
	file1 := "./patient.csv"
	nofCohortAges := 10
	patients, _ := lib.ParseTriNetXPatientData(file1, nofCohortAges)
	file2 := "./diagnosis.csv"
	file3 := "./icd10cm_tabular_2022.xml"
	level := 0
	analysisMaps := lib.InitializeIcd10AnalysisMapsFromXML(file3, level)
	lib.ParseTrinetXPatientDiagnoses(file2, "", patients, analysisMaps, map[string]string{})
	fmt.Println("First 5 patients: ")
	ctr := 0
	for _, patient := range patients.PIDMap {
		if ctr == 5 {
			break
		}
		if len(patient.Diagnoses) > 0 {
			ctr++
			fmt.Println(patient)
		}
	}
}

func TestInitCohortsWithFakePatients(t *testing.T) {
	n := 100
	patients := []*lib.Patient{}
	for i := 0; i < n; i++ {
		p := lib.Patient{
			PID:       i,
			PIDString: fmt.Sprint(i),
			YOB:       1900 + i,
			CohortAge: 0,
			Sex:       0,
			Diagnoses: nil,
		}
		if p.YOB >= 1950 {
			p.CohortAge = 1
		}
		d1 := lib.Diagnosis{PID: i, DID: 0, Date: lib.DiagnosisDate{Year: 2019, Day: 26, Month: 8}} // smoking
		d2 := lib.Diagnosis{PID: i, DID: 1, Date: lib.DiagnosisDate{Year: 2020, Day: 26, Month: 8}} // cancer1
		d3 := lib.Diagnosis{PID: i, DID: 2, Date: lib.DiagnosisDate{Year: 2021, Day: 26, Month: 8}} // drinking
		d4 := lib.Diagnosis{PID: i, DID: 3, Date: lib.DiagnosisDate{Year: 2022, Day: 26, Month: 8}} // cancer2
		p.Diagnoses = []*lib.Diagnosis{&d1, &d2, &d3, &d4}
		// try to show a strong link between smoking->cancer1 and drinking->cancer2
		patients = lib.AppendPatient(patients, &p)
	}
	for i := n; i < 2*n; i++ {
		p := lib.Patient{
			PID:       i,
			PIDString: fmt.Sprint(i),
			YOB:       1900 + i - n,
			CohortAge: 0,
			Sex:       1,
			Diagnoses: nil,
		}
		if p.YOB >= 1950 {
			p.CohortAge = 1
		}
		d1 := lib.Diagnosis{PID: i, DID: 0, Date: lib.DiagnosisDate{Year: 2019, Day: 26, Month: 8}} // smoking
		d2 := lib.Diagnosis{PID: i, DID: 1, Date: lib.DiagnosisDate{Year: 2020, Day: 26, Month: 8}} // cancer1
		d3 := lib.Diagnosis{PID: i, DID: 2, Date: lib.DiagnosisDate{Year: 2021, Day: 26, Month: 8}} // drinking
		d4 := lib.Diagnosis{PID: i, DID: 3, Date: lib.DiagnosisDate{Year: 2022, Day: 26, Month: 8}} // cancer2
		p.Diagnoses = []*lib.Diagnosis{&d1, &d2, &d3, &d4}
		patients = lib.AppendPatient(patients, &p)
	}
	for i := 2 * n; i < 3*n; i++ {
		p := lib.Patient{
			PID:       i,
			PIDString: fmt.Sprint(i),
			YOB:       1900 + i - 2*n,
			CohortAge: 0,
			Sex:       0,
			Diagnoses: nil,
		}
		if p.YOB >= 1950 {
			p.CohortAge = 1
		}
		//d1 := Diagnosis{PID: i, DID: 0, Date: DiagnosisDate{Year: 2019, Day: 26, Month: 8},} no smoking
		d2 := lib.Diagnosis{PID: i, DID: 1, Date: lib.DiagnosisDate{Year: 2020, Day: 26, Month: 8}}
		//d3 := Diagnosis{PID: i, DID: 2, Date: DiagnosisDate{Year: 2020, Day: 26, Month: 8},} no drinking
		d4 := lib.Diagnosis{PID: i, DID: 3, Date: lib.DiagnosisDate{Year: 2021, Day: 26, Month: 8}}
		p.Diagnoses = []*lib.Diagnosis{}
		// small nr of people get cancer1 without smoking
		if p.YOB >= 1925 && p.YOB <= 1930 {
			p.Diagnoses = append(p.Diagnoses, &d2)
		}
		if p.YOB >= 1980 && p.YOB <= 1985 {
			p.Diagnoses = append(p.Diagnoses, &d2)
		}
		if p.YOB >= 1945 && p.YOB <= 1950 {
			p.Diagnoses = append(p.Diagnoses, &d4)
		}
		if p.YOB >= 1990 && p.YOB <= 1995 {
			p.Diagnoses = append(p.Diagnoses, &d4)
		}
		patients = lib.AppendPatient(patients, &p)
	}
	for i := 3 * n; i < 4*n; i++ {
		p := lib.Patient{
			PID:       i,
			PIDString: fmt.Sprint(i),
			YOB:       1920 + i - 3*n,
			CohortAge: 0,
			Sex:       1,
			Diagnoses: nil,
		}
		if p.YOB >= 1970 {
			p.CohortAge = 1
		}
		//d1 := Diagnosis{PID: i, DID: 0, Date: DiagnosisDate{Year: 2019, Day: 26, Month: 8},}
		d2 := lib.Diagnosis{PID: i, DID: 1, Date: lib.DiagnosisDate{Year: 2020, Day: 26, Month: 8}}
		//d3 := Diagnosis{PID: i, DID: 2, Date: DiagnosisDate{Year: 2020, Day: 26, Month: 8},}
		d4 := lib.Diagnosis{PID: i, DID: 3, Date: lib.DiagnosisDate{Year: 2021, Day: 26, Month: 8}}
		p.Diagnoses = []*lib.Diagnosis{}
		// small nr of people get cancer1 without smoking
		if p.YOB >= 1925 && p.YOB <= 1930 {
			p.Diagnoses = append(p.Diagnoses, &d2)
		}
		if p.YOB >= 1980 && p.YOB <= 1985 {
			p.Diagnoses = append(p.Diagnoses, &d2)
		}
		if p.YOB >= 1945 && p.YOB <= 1950 {
			p.Diagnoses = append(p.Diagnoses, &d4)
		}
		if p.YOB >= 1990 && p.YOB <= 1995 {
			p.Diagnoses = append(p.Diagnoses, &d4)
		}
		patients = lib.AppendPatient(patients, &p)
	}
	pMap := map[int]*lib.Patient{}
	pidMap := map[string]int{}
	ctr := 0
	for _, patient := range patients {
		ctr++
		pMap[patient.PID] = patient
		pidMap[patient.PIDString] = patient.PID
	}
	PMap := &lib.PatientMap{
		PIDStringMap: pidMap,
		Ctr:          ctr,
		PIDMap:       pMap,
		MaleCtr:      ctr,
		FemaleCtr:    0,
	}
	cohorts := lib.InitCohorts(PMap, 2, 1, 4)
	fmt.Println("Printing cohorts")
	for _, cohort := range cohorts {
		cohort.Log(4)
	}
	cohort := lib.MergeCohorts(cohorts)
	cohort.Log(4)
	// Test building trajectories
	icd10Map := map[int]lib.Icd10Entry{
		0: {
			Name:       "Smoking",
			Categories: [6]string{"cat1", "cat2", "cat3", "None", "None", "None"},
			Level:      3,
		},
		1: {
			Name:       "Lung cancer",
			Categories: [6]string{"cat1", "cat2", "None", "None", "None", "None"},
			Level:      2,
		},
		2: {Name: "Drinking",
			Categories: [6]string{"cat1", "cat2", "cat3", "cat4", "cat5", "None"},
			Level:      5,
		},
		3: {Name: "Liver cancer",
			Categories: [6]string{"cat1", "None", "None", "None", "None", "None"},
			Level:      1,
		},
	}
	exp := &lib.Experiment{
		NofAgeGroups:      2,
		Level:             0,
		NofDiagnosisCodes: 4,
		DxDRR:             lib.MakeDxDRR(4),
		DxDPatients:       lib.MakeDxDPatients(4),
		DPatients:         cohort.DPatients,
		Name:              "exp1",
		Cohorts:           cohorts,
		Icd10Map:          icd10Map,
		Trajectories:      nil,
	}
	exp.InitRR(0.5, 5.0, 10)
	fmt.Println("Relative risk ratios: [")
	for _, rr := range exp.DxDRR {
		fmt.Print(rr, ", ")
	}
	fmt.Println("...]")
	trajectories := exp.BuildTrajectories(5, 3, 2, 1, 5, 1.0, []lib.TrajectoryFilter{})
	fmt.Println("Collected ", len(trajectories), " trajectories.")
	for _, traj := range trajectories {
		lib.LogTrajectory(traj, exp)
	}
	exp.PrintTrajectoriesToFile("./output")
	// Output should be:
	// Building patient trajectories...
	// Selecting diagnosis pairs for building trajectories...
	// Found  4  suitable diagnosis pairs.
	// Found  5  trajectories.
	// Collected  5  trajectories.
	// Smoking -- 200 --> Lung cancer
	// Smoking -- 200 --> Drinking -- 200 --> Liver cancer
	// Smoking -- 200 --> Drinking
	// Smoking -- 200 --> Liver cancer
	// Drinking -- 200 --> Liver cancer
}
