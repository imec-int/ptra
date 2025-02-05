package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/imec-int/ptra/pkg/trinetx"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
)

/*
Ptra is a tool for patient trajectory analysis.

Usage:
	ptra pfile ifile dfile path [flags]

Example:
	ptra ICD10 patient.csv icd10cm_tabular_2022.xml diagnosis.csv ./MIBC_tfiltered/ --nofAgeGroups 10 --lvl 2
	--maxYears 5 --minYears 0.001 --minPatients 50 --maxTrajectoryLength 5 --minTrajectoryLength 3 --name MICB_tfiltered
	--ICD9ToICD10File ICD_9_to_10.json --iter 400 --RR 1.0 --tumorInfo tumor.csv --saveRR MIBC_tfiltered.csv --cluster
	--mclPath /home/caherzee/tools/mcl/ --clusterGranularities 40,60,80,100 --pfilters "MIBC" --tumorInfo tumor.csv
	--tfilters "bc" --treatmentInfo treatments.csv

The flags are:

--nofAgeGroups nr
	The number of age groups to consider when dividing the population into cohorts based on age. The tool automatically
	detects the oldest and youngest patients from input. The number of age groups is used to divide the minumum birth
	year and maximum birth year into age ranges. E.g. min birth year: 1900, max birth year: 2020, and nr of age groups:
	10, will create 10 cohorts where age ranges from [0,12],[12,22],...[108,120].
--lvl nr
	Sets the ICD10 level for the diagnosis input [0-6]. The tool maps all ICD10 codes to a medical meaningful term based on
	this chosen level. ICD10 codes of lower levels may be combined into the same code of a higher level. E.g. A00.0
	Cholera due to Vibrio cholerae 01, biovar cholerae and A00.1 Cholera due to Vibrio cholerae 01, biovar eltor are lvl
	3 codes and may be collapsed to A00 Cholera in lvl 2, or A00-A09 Intestinal infectious diseases in lvl 1, or A00-B99
	Certain infectious and parasitic diseases in lvl 0.
--minPatients nr
	Sets the minimum required number of patients in a trajectory.
--maxYears nr
	Sets the maximum number of years between subsequent diagnoses to be considered for inclusion in a trajectory. E.g.
	0.5 for half a year.
--minYears nr
	Sets the minimum number of years between subsequent diagnoses to be considered for inclusion in a trajectory. E.g.
	0.5 for half a year.
--maxTrajectoryLength nr
	Sets the maximum length of trajectories to be included in the output. E.g. 5 for trajectories with maximum 5
	diagnoses.
--minTrajectoryLength nr
	Sets the minimum length of trajectories to be included in the output. E.g. 3 for trajectories with minimum 3
	diagnoses.
--name string
	Sets the name of the experiment. This name is used to generate names for output files.
--ICD9ToICD10File file
	A json file that provides a mapping from ICD9 to ICD10 codes. The input may be mixed ICD9 and ICD10 codes. With this
	mapping, the tool can automatically convert all diagnosis codes to ICD10 codes for analysis.
--cluster
	If this flag is passed, the computed trajectories are clustered and the clusters are outputted to file.
--mclPath
	Sets the path where the mcl binaries can be found.
--iter nr
	Sets the number of iterations to be used in the sampling experiments for calculating relative risk ratios. If iter
	is 400, the calculated p-values are within 0.05 of the true p-values. For iter = 10000, the true p-values are within
	0.01 of the true p-values. The higher the number of iterations, the higher the runtime.
--saveRR file
	Save the RR matrix, a matrix that represents the RR calculated from the population for each possible combination of
	ICD10 diagnosis pairs. This matrix can be loaded in other ptra runs to avoid recalculating the RR scores. This can
	be useful if parameters want to be explored that do not impact the RR calculation itself. Only iter, maxYears and
	minYears, and filters influence RR calculation. Variations of other parameters for constructing trajectories from RR
	scores, such as maxTrajectoryLenght, minTrajectoryLength, minPatients, RR etc might be explored in other runs.
--loadRR file
	Load the RR matrix from file. Such a file must be created by a previous run of ptra with the --saveRR flag.
--pfilters age70+ | age70- | male | female | Ta | T0 | Tis | T1 | T2 | T3 | T4 | N0 | N1 | N2 | N3 | M0 | M1 |NMIBC | MIBC | mUC
	A list of filters for selecting patients from whitch to derive trajectories.
--tumorInfo file
	A file with information about patients and their tumors. This file contains annotations about the stage of the
	bladder cancer at a specific time. Cf. TriNetX tumor table. This information is used by filters.
--tfilters neoplasm | bc
	A list of filters for reducing the output of trajectories. E.g. neoplasm only outputs trajectories where there is at
	least one diagnosis related to cancer. bc only outputs trajectories where one diagnosis is (assuming) related to
	bladder cancer.
--treatmentInfo file
	A file with information about patients and their treatments, e.g. MVAC,radical cystectomy, etc. If this file is
	passed, the treatments will be used as diagnostic codes to calculated trajectories.
*/

const (
	programVersion = 0.1
	programName    = "ptra"
)

func programMessage() string {
	return fmt.Sprint(programName, " version ", programVersion, " compiled with ", runtime.Version())
}

const ptraHelp = "\nptra parameters:\n" +
	"ptra patientInfoFile diagnosisInfoFile diagnosesFile outputPath \n" +
	"[--nofAgeGroups nr]\n" +
	"[--lvl nr]\n" +
	"[--minPatients nr]\n" +
	"[--maxYears nr]\n" +
	"[--minYears nr]\n" +
	"[--maxTrajectoryLength nr]\n" +
	"[--minTrajectoryLength nr]\n" +
	"[--name string]\n" +
	"[--ICD9ToICD10File file]\n" +
	"[--cluster]\n" +
	"[--mclPath string]\n" +
	"[--iter nr]\n" +
	"[--saveRR file]\n" +
	"[--loadRR file]\n" +
	"[--pfilters age70+ | age70- | male | female | Ta | T0 | Tis | T1 | T2 | T3 | T4 | N0 | N1 | N2 | N3 | M0 | M1 |" +
	"NMIBC | MIBC | mUC ]\n" +
	"[--tumorInfo file]\n" +
	"[--tfilters neoplasm | bc]\n" +
	"[--treatmentInfo file]\n" +
	"[--nrOfThreads nr]\n"

func parseFlags(flags flag.FlagSet, requiredArgs int, help string) {
	if len(os.Args) < requiredArgs {
		fmt.Fprintln(os.Stderr, "Incorrect number of parameters.")
		fmt.Fprint(os.Stderr, help)
		os.Exit(1)
	}
	flags.SetOutput(ioutil.Discard)
	if err := flags.Parse(os.Args[requiredArgs:]); err != nil {
		x := 0
		if err != flag.ErrHelp {
			fmt.Fprint(os.Stderr, err)
		}
		fmt.Fprint(os.Stderr, help)
		os.Exit(x)
	}
	if flags.NArg() > 0 {
		fmt.Fprint(os.Stderr, "Cannot parse remaining parameters:", flags.Args())
		fmt.Fprint(os.Stderr, help)
		os.Exit(1)
	}
}

func getFileName(s, help string) string {
	switch s {
	case "-h", "--h", "-help", "--help":
		fmt.Fprint(os.Stderr, help)
		os.Exit(1)
	}
	return s
}

func main() {
	var params = trinetx.ExperimentParams{}
	var flags flag.FlagSet

	// extract ExperimentParams from command line params
	flags.IntVar(&params.NofAgeGroups, "nofAgeGroups", 6, "The population data is divided in cohorts in"+
		"terms of age groups to calculate relative risk ratios of diagnosis pairs. This parameters configures how"+
		"many age groups to use")
	flags.IntVar(&params.NrOfThreads, "nrOfThreads", 0, "The number of threads ptra uses.")
	flags.IntVar(&params.Lvl, "lvl", 3, "Diagnosis codes are organised in a hierarchy of diagnosis "+
		"descriptors. The level says which descriptor in the hiearchy to use for trajectory building.")
	flags.Float64Var(&params.MaxYears, "maxYears", 5.0, "The maximum number of years between diagnosis "+
		"A and B to consider the diagnosis pair A->B in a trajectory.")
	flags.Float64Var(&params.MinYears, "minYears", 0.5, "The minimum number of years between diagnisis "+
		"A and B to consider the diagnosis pair A->B in a trajectory.")
	flags.IntVar(&params.MinPatients, "minPatients", 1000, "The minimum number of patients for the last "+
		"diagnosis in a trajectory")
	flags.IntVar(&params.MaxTrajectoryLength, "maxTrajectoryLength", 5, "The maximum number of diagnoses"+
		" in a trajectory")
	flags.IntVar(&params.MinTrajectoryLength, "minTrajectoryLength", 3, "The minimum number of "+
		"diagnoses in a trajectory")
	flags.StringVar(&params.Name, "name", "exp1", "The name of the run. This is used to generate the "+
		"names of the output files.")
	flags.StringVar(&params.ICD9ToICD10File, "ICD9ToICD10File", "", "A json file that maps ICD9 to "+
		"ICD10 codes.")
	flags.BoolVar(&params.Cluster, "cluster", false, "Cluster the trajectories using MCL and output "+
		"the results")
	flags.StringVar(&params.ClusterGranularities, "clusterGranularities", "40,60,80,100", "The "+
		"granularities used for the mcl clustering step.") // recommended 14,20,40,60
	flags.IntVar(&params.Iter, "iter", 10000, "The minimum number of sampling iterations "+
		"diagnosis in a trajectory")
	flags.Float64Var(&params.RR, "RR", 1.0, "The minimum RR score for considering pairs.")
	flags.StringVar(&params.SaveRR, "saveRR", "", "Save the RR matrix to a file so it can be loaded for "+
		"later runs")
	flags.StringVar(&params.LoadRR, "loadRR", "", "Load the RR matrix from a given file instead of "+
		"calculating it from scratch.")
	flags.StringVar(&params.PFilters, "pfilters", "id", "A list of pfilters to restrict analysis on specific "+
		"patients.")
	flags.StringVar(&params.TumorInfo, "tumorInfo", "", "A file with information about the tumor stages.")
	flags.StringVar(&params.TreatmentInfo, "treatmentInfo", "", "A file with information about patient cancer stages.")
	flags.StringVar(&params.TFilters, "tfilters", "id", "A list of pfilters to restrict output of trajectories")

	// parse optional arguments
	parseFlags(flags, 5, ptraHelp)

	// parse required arguments
	params.PatientInfo = getFileName(os.Args[1], ptraHelp)
	params.DiagnosisInfo = getFileName(os.Args[2], ptraHelp)
	params.PatientDiagnoses = getFileName(os.Args[3], ptraHelp)
	params.OutputPath, _ = filepath.Abs(getFileName(os.Args[4], ptraHelp))
	fmt.Println("Output path: ", params.OutputPath)

	// build an output command line
	var command bytes.Buffer
	fmt.Fprint(&command, os.Args[0], " ", params.PatientInfo, " ", params.DiagnosisInfo, " ", params.PatientDiagnoses,
		" ", params.OutputPath)
	fmt.Fprint(&command, " --nofAgeGroups ", params.NofAgeGroups)
	fmt.Fprint(&command, " --lvl ", params.Lvl)
	fmt.Fprint(&command, " --maxYears ", params.MaxYears)
	fmt.Fprint(&command, " --minYears ", params.MinYears)
	fmt.Fprint(&command, " --minPatients ", params.MinPatients)
	fmt.Fprint(&command, " --maxTrajectoryLength ", params.MaxTrajectoryLength)
	fmt.Fprint(&command, " --minTrajectoryLength ", params.MinTrajectoryLength)
	fmt.Fprint(&command, " --name ", params.Name)
	fmt.Fprint(&command, " --ICD9ToICD10File ", params.ICD9ToICD10File)
	fmt.Fprint(&command, " --iter ", params.Iter)
	fmt.Fprint(&command, " --RR ", params.RR)
	fmt.Fprint(&command, " --tumorInfo ", params.TumorInfo)
	fmt.Fprint(&command, " --treatmentInfo ", params.TreatmentInfo)

	if params.SaveRR != "" {
		fmt.Fprint(&command, " --saveRR ", params.SaveRR)
	}

	if params.LoadRR != "" {
		fmt.Fprint(&command, " --loadRR ", params.LoadRR)
	}

	if params.Cluster {
		fmt.Fprint(&command, " --cluster")
		fmt.Fprint(&command, " --clusterGranularities ", params.ClusterGranularities)
	}

	fmt.Fprint(&command, " --pfilters ", params.PFilters)
	fmt.Fprint(&command, " --tfilters ", params.TFilters)

	if params.NrOfThreads > 0 {
		fmt.Fprint(&command, " --nrOfThreads ", params.NrOfThreads)
	}

	err := trinetx.Run(&params)
	if err != nil {
		panic(err)
	}
}
