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

package lib

import (
	"fmt"
	"github.com/imec-int/ptra/lib/utils"
	"io"
	"os"
	"path/filepath"
	"strconv"
)

// Plotting of trajectories

// LogTrajectory prints a trajectory to standard output.
func LogTrajectory(t *Trajectory, exp *Experiment) {
	j := 0
	for i, d := range t.Diagnoses {
		dName := exp.Icd10Map[d].Name
		fmt.Print(dName)
		if i != len(t.Diagnoses)-1 {
			fmt.Print(" -- ", t.PatientNumbers[j], " --> ")
		}
		j++
	}
	fmt.Println(" ")
}

// printTrajectoriesToTabFile prints a human-readable representation of trajectories to a tab file. Per trajectory, it
// prints two lines. A first line is a list of medical terms for diagnoses in the trajectory (in order of occurrence):
// term1 tab term2 tab ... termn. The second line lists the number of patients for each transition in the trajectory:
// nr1->2 tab nr2->3 tab ... nrn-1->n.
func printTrajectoriesToTabFile(trajectories []*Trajectory, icd10Map map[int]Icd10Entry, name string) {
	file, err := os.Create(name)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			panic(err)
		}
	}()
	for _, trajectory := range trajectories {
		nodes := trajectory.Diagnoses
		labels := trajectory.PatientNumbers
		var line string
		for i, node := range nodes {
			if i < len(nodes)-1 {
				line = fmt.Sprintf("%s%s\t", line, icd10Map[node].Name)
			} else {
				line = fmt.Sprintf("%s%s\n", line, icd10Map[node].Name)
			}
		}
		fmt.Fprintf(file, line)
		line = ""
		for i, label := range labels {
			if i < len(labels)-1 {
				line = fmt.Sprintf("%s%d\t", line, label)
			} else {
				line = fmt.Sprintf("%s%d\n", line, label)
			}
		}
		fmt.Fprintf(file, line)
	}
}

// printPairsToTableFile prints the diagnosis pairs and the associated relative risks scores in a human-readable format
// to a tab file. For each diagnosis pair, it prints one line that lists the medical terms for the diagnoses and the
// relative risk score: term1 tab term2 tab RR.
func printPairsToTabFile(exp *Experiment, name string) {
	pairs := exp.Pairs
	file, err := os.Create(name)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			panic(err)
		}
	}()
	for _, pair := range pairs {
		fmt.Fprintf(file, "%s\t%s\t%s\n", exp.Icd10Map[pair.First].Name, exp.Icd10Map[pair.Second].Name,
			strconv.FormatFloat(exp.DxDRR[pair.First][pair.Second], 'E', -1, 64))
	}
}

// convertTrajectoriesToGraph converts an experiment's trajectories to an adjacency matrix graph representation. The
// function returns a list of nodes and an adjacency matrix with edge connections as result values.
func convertTrajectoriesToGraph(exp *Experiment) ([]int, [][][]int) {
	trajectories := exp.Trajectories
	am := make([][][]int, exp.NofDiagnosisCodes)
	for i, _ := range am {
		am[i] = make([][]int, exp.NofDiagnosisCodes)
	}
	nodes := []int{}
	for _, traj := range trajectories {
		// collect nodes
		for _, d := range traj.Diagnoses {
			if !utils.MemberInt(d, nodes) {
				nodes = append(nodes, d)
			}
		}
		// collect edges
		i := 0
		first := traj.Diagnoses[i]
		for j := 1; j < len(traj.Diagnoses); j++ {
			second := traj.Diagnoses[j]
			n := traj.PatientNumbers[i]
			if am[first][second] != nil {
				if !utils.MemberInt(n, am[first][second]) {
					am[first][second] = append(am[first][second], n)
				}
			} else {
				am[first][second] = []int{n}
			}
			i = j
			first = second
		}
	}
	return nodes, am
}

func printTrajectory(trajectory *Trajectory, exp *Experiment, w io.Writer) {
	TID := trajectory.ID
	diagnoses := trajectory.Diagnoses
	icd10Map := exp.Icd10Map
	d1 := trajectory.Diagnoses[0]

	// print nodes
	for _, node := range diagnoses {
		icd10 := icd10Map[node]
		fmt.Fprintf(w, "\tnode [\n\t\tid %d\n\t\tlabel \"%s\"\n\t", node, icd10.Name)
		fmt.Fprintf(w, "\tlevel %d\n", icd10.Level)
		for idx, cat := range icd10.Categories {
			if cat == "NONE" {
				break
			}
			fmt.Fprintf(w, "\t\tcat%d \"%s\"\n", idx, cat)
		}
		fmt.Fprintf(w, "\t]\n")
	}

	// print edges
	tlen := len(diagnoses) - 1
	for idx := 0; idx < tlen; idx++ {
		source := diagnoses[idx]
		target := diagnoses[idx+1]
		d2 := trajectory.Diagnoses[idx]
		patients := trajectory.PatientNumbers[idx]
		RR := strconv.FormatFloat(exp.DxDRR[d1][d2], 'f', 2, 64)
		fmt.Fprintf(w, fmt.Sprintf("\tedge [\n\t\ttid %d\n\t\ttlen %d\n\t\ttidx %d\n\t\tsource %d\n\t\ttarget %d\n\t\tpatients %d\n\t\tRR \"%s\"\n\t]\n", TID, tlen, idx, source, target, patients, RR))
	}
}

// printTrajectories plots all of an experiment's trajectories as a single graph to a GML file. The nodes
// in the graph are the medical terms for the diagnoses that make up the trajectories. The edges are derived from the
// transitions between diagnoses in the trajectories.
func printTrajectories(exp *Experiment, name string) {
	file, err := os.Create(name)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			panic(err)
		}
	}()

	// open graph
	fmt.Fprintf(file, "graph [\n\tdirected 1\n\tmultigraph 1\n")
	for _, t := range exp.Trajectories {
		printTrajectory(t, exp, file)
	}
	// close graph
	fmt.Fprintf(file, "]\n")
}

// printIndividualTrajectories prints each trajectory as a separate subgraph to the same GML output file.
func printIndividualTrajectories(exp *Experiment, name string) {
	file, err := os.Create(name)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			panic(err)
		}
	}()

	for _, t := range exp.Trajectories {
		// new graph per trajectory
		fmt.Fprintf(file, "graph [\n\tdirected 1\n\tmultigraph 1\n")
		printTrajectory(t, exp, file)
		// close graph
		fmt.Fprintf(file, "]\n")
	}
}

// PrintTrajectoriesToFile outputs an experiment's calculated trajectories to file in multiple formats:
// - A tab file containing trajectories as lists of medical terms and lists of numbers of patients for each transition
// - A tab file containing all disease pairs and their relative risk scores (medical terms + float for RR)
// - A GML file with one graph representing all trajectories
// - A GML file where each trajectory is represented as an individual subgraph
func (exp *Experiment) PrintTrajectoriesToFile(path string) {
	// print the trajectories to file
	// create a file where all trajectories are separate graphs
	// create a file where all trajectories are combined into 1 graph
	// create a file that just has each trajectory as a tab seperated list of disease codes
	os.Mkdir(path, 0700)
	tabFileName := filepath.Join(path, fmt.Sprintf("%s-trajectories.tab", exp.Name))
	printTrajectoriesToTabFile(exp.Trajectories, exp.Icd10Map, tabFileName)
	tabFileName2 := filepath.Join(path, fmt.Sprintf("%s-pairs.tab", exp.Name))
	printPairsToTabFile(exp, tabFileName2)
	graphFileName := filepath.Join(path, fmt.Sprintf("%s-trajectories-merged-graph.gml", exp.Name))
	printTrajectories(exp, graphFileName)
	graphsFileName := filepath.Join(path, fmt.Sprintf("%s-trajectories-individual-graphs.gml", exp.Name))
	printIndividualTrajectories(exp, graphsFileName)
}

// collectClusters returns a map from cluster ID to a set of trajectories that belong to that cluster
func collectClusters(exp *Experiment) map[int][]*Trajectory {
	clusters := map[int][]*Trajectory{}
	for _, t := range exp.Trajectories {
		if _, ok := clusters[t.Cluster]; ok {
			clusters[t.Cluster] = append(clusters[t.Cluster], t)
		} else {
			clusters[t.Cluster] = []*Trajectory{t}
		}
	}
	return clusters
}

// PrintClusteredTrajectoriesToFile plots the trajectories of an experiment to a tab file, including for each trajectory
// information about the cluster a trajectory belongs to. For each trajectory it prints 3 lines:
// - A line with the cluster ID and the trajectory ID: CID: \tab nr \tab TID: \tab nr.
// - A list of medical terms for the diagnoses: term1 \tab term2 ...\tab termn.
// - A list of patient numbers for the transitions between diagnosis pairs: nr1->2 \tab nr2->3 ...\tab nrn-1->n.
func PrintClusteredTrajectoriesToFile(exp *Experiment, name string) {
	// plots a line with cluster ID, trajectory ID
	// plots a line with trajectory
	// plots a line with trajectory labels (= nr of patients)
	file, err := os.Create(name)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			panic(err)
		}
	}()
	clusters := collectClusters(exp)
	for i := 0; i < len(clusters); i++ {
		c := clusters[i]
		// print out metrics of the c
		ageMean, stdev, ageEOIMean, stdev2, mCtr, fCtr := MetricsFromTrajectories(c)
		line := fmt.Sprintf("CID:\t%d\tMean Age:\t%s\tStdev:\t%s\tMean Age EOI:\t%s\tStdev:\t%s\tMales:\t%d\tFemales:\t%d\tTrajectories:\t%d\n",
			i,
			strconv.FormatFloat(ageMean, 'f', 2, 64),
			strconv.FormatFloat(stdev, 'f', 2, 64),
			strconv.FormatFloat(ageEOIMean, 'f', 2, 64),
			strconv.FormatFloat(stdev2, 'f', 2, 64), mCtr, fCtr, len(c))
		fmt.Fprintf(file, line)
		line = ""
		// print the trajectories to tab file
		for _, trajectory := range c {
			nodes := trajectory.Diagnoses
			labels := trajectory.PatientNumbers
			//print cluster and trajectory IDs
			line = fmt.Sprintf("%sCID:\t%d\tTID:\t%d\n", line, i, trajectory.ID)
			fmt.Fprintf(file, line)
			line = ""
			// print trajectory
			for i, node := range nodes {
				separator := "\t"
				if i == len(nodes)-1 {
					separator = "\n"
				}
				line += fmt.Sprintf("%s%s", exp.Icd10Map[node].Name, separator)

			}
			fmt.Fprintf(file, line)
			line = ""
			for i, label := range labels {
				if i < len(labels)-1 {
					line = fmt.Sprintf("%s%d\t", line, label)
				} else {
					line = fmt.Sprintf("%s%d\n", line, label)
				}
			}
			fmt.Fprintf(file, line)
			line = ""
		}
	}
}

// PrintClustersToCSVFiles prints the experiment clusters to a CSV file. It creates two output files:
// - A CSV file with patient information. The header is: PID,AgeEOI,Sex,PIDString. This represents: patient analysis id,
// age at which the event of interest occurred, sex, and the TriNetX patient id.
// - A CSV file with cluster information. The header is: PID,CID,TID,Age. This represents: patient id, cluster id,
// trajectory id, and age of the patient when matching the trajectory.
func PrintClustersToCSVFiles(exp *Experiment, pName, cName string) {
	// print the patients information for this cluster to a CSV file containing:
	// PID, Age, AgeEOI, Sex, PIDString
	pFile, err := os.Create(pName)
	if err != nil {
		panic(err)
	}
	// print header
	fmt.Fprintf(pFile, "PID,AgeEOI,Sex,PIDString\n")
	pSeen := map[int]bool{}
	for _, t := range exp.Trajectories {
		ps := t.Patients
		for _, p := range ps[len(ps)-1] {
			if _, ok := pSeen[p.PID]; !ok {
				pSeen[p.PID] = true
				ageEOI := AgeAtEOI(p)
				var sex string
				if p.Sex == Male {
					sex = "M"
				} else {
					sex = "F"
				}
				fmt.Fprintf(pFile, "%d,%d,%s,%s\n", p.PID, ageEOI, sex, p.PIDString)
			}
		}
	}
	// close file
	if err := pFile.Close(); err != nil {
		panic(err)
	}
	// print the cluster information to a CSV file containing:
	// PID,CID,TID
	cFile, err := os.Create(cName)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := cFile.Close(); err != nil {
			panic(err)
		}
	}()
	// print header
	fmt.Fprintf(cFile, "PID,CID,TID,Age\n")
	for _, t := range exp.Trajectories {
		ps := t.Patients
		for _, p := range ps[len(ps)-1] {
			age := AgeAtDiagnosis(p, t.Diagnoses[len(t.Diagnoses)-1])
			fmt.Fprintf(cFile, "%d,%d,%d,%d\n", p.PID, t.Cluster, t.ID, age)
		}
	}
}
