package framework

import (
	"fmt"
	"math"
	"os"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/olekukonko/tablewriter"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// err string, for tight column display which is not interested in the full error
	errStr = "err"
)

// TrackedLabel defines resources to track and monitor during the test
type TrackedLabel struct {
	Namespace string
	Label     metav1.LabelSelector
}

// GrafanaPanel identifies a target Panel to be saved at the end of the test
type GrafanaPanel struct {
	Filename  string
	Dashboard string
	Panel     int
}

// Resource is used to store seen tracked resources during the test
type Resource struct {
	Namespace     string
	PodName       string
	ContainerName string
}

// DataHandle is the handle to specific resource information during a test
type DataHandle struct {
	// Test start time
	TestStartTime time.Time

	// Prometheus and Grafana handles
	PromHandle *Prometheus
	GrafHandle *Grafana

	// Result output files, allows setting multiple descriptors to write results
	ResultsOut []*os.File

	// Defines the resources to keep track across iterations
	TrackAppLabels []TrackedLabel
	// Defines the panels to save upon WrapUp invocation
	GrafanaPanelsToSave []GrafanaPanel

	// Set of names to track exact resources that were observed at any point in time
	// during the test
	SeenResources map[Resource]bool

	// Scale iterations data
	Iterations  int
	ItStartTime []time.Time
	ItEndTime   []time.Time
}

// NewDataHandle Returns an initialized scale data
func NewDataHandle(pHandle *Prometheus, gHandle *Grafana, trackResources []TrackedLabel, saveDashboards []GrafanaPanel) *DataHandle {
	return &DataHandle{
		TestStartTime:       time.Now(),
		PromHandle:          pHandle,
		GrafHandle:          gHandle,
		TrackAppLabels:      trackResources,
		GrafanaPanelsToSave: saveDashboards,
		Iterations:          0,
		ItStartTime:         []time.Time{},
		ItEndTime:           []time.Time{},
		SeenResources:       map[Resource]bool{},
		ResultsOut:          []*os.File{},
	}
}

// WrapUp is a callback to execute after the test is done.
// Will output all iteration results, compute some relative usages between iterations
// and save grafana dashboards if any are to be saved
func (sd *DataHandle) WrapUp() {
	for _, f := range sd.ResultsOut {
		sd.OutputTestResults(f)
	}

	if sd.GrafHandle != nil {
		for _, panel := range sd.GrafanaPanelsToSave {
			minutesElapsed := int(math.Ceil(time.Since(sd.TestStartTime).Minutes()))
			err := sd.GrafHandle.PanelPNGSnapshot(panel.Dashboard, panel.Panel, minutesElapsed, Td.GetTestFile(panel.Filename))
			fmt.Printf("%v", err)
		}
	}
}

// IterateTrackedPods is a Helper to iterate pods selected by the Tracked labels
func (sd *DataHandle) IterateTrackedPods(f func(pod *corev1.Pod)) {
	for _, trackLabel := range sd.TrackAppLabels {
		podsForLabel, err := Td.GetPodsForLabel(trackLabel.Namespace, trackLabel.Label)
		Expect(err).To(BeNil())

		for idx := range podsForLabel {
			f(&podsForLabel[idx])
		}
	}
}

// Iterate wrapper to loop and track various resources across iterations (time, mem consumption, iteration num, etc.)
func (sd *DataHandle) Iterate(f func()) {
	for {
		sd.ItStartTime = append(sd.ItStartTime, time.Now())
		f()
		sd.ItEndTime = append(sd.ItEndTime, time.Now())

		diff := sd.ItEndTime[sd.Iterations].Sub(sd.ItStartTime[sd.Iterations])

		fmt.Printf("-- Successfully completed iteration %d - took %v\n", sd.Iterations, diff)
		sd.OutputIteration(sd.Iterations, os.Stdout)
		fmt.Println("--------")

		// Increase iterations done
		sd.Iterations++
	}
}

// shortenResName shortens the resource name for better columns display
func shortenResName(res Resource) string {
	// examples
	// osm-cont..3452/osm-con..
	// osm-prom..9284/prometh..
	var podShortened, contShortened string
	if len(res.PodName) > 14 { // 12 + len("..")
		podShortened = fmt.Sprintf("%s..%s", res.PodName[:7], res.PodName[len(res.PodName)-4:])
	} else {
		podShortened = res.PodName
	}

	if len(res.ContainerName) > 7 { // 5 + len("..")
		contShortened = fmt.Sprintf("%s..", res.ContainerName[:4])
	} else {
		contShortened = res.ContainerName
	}

	return fmt.Sprintf("%s/%s", podShortened, contShortened)
}

// Given two floats, return the signed relative percentual difference as a string
func percentDiff(a float64, b float64) string {
	s := ""
	if a > b {
		s = fmt.Sprintf("+%.2f%%", (((a / b) - 1) * 100))
	} else if a < b {
		s = fmt.Sprintf("-%.2f%%", ((1 - (a / b)) * 100))
	} else {
		s = "0%"
	}
	return s
}

// Output functions/boiler plate for information display below.

// OutputIteration prints on file <f> the results of iteration <iterNumber>
// The CpuAvg is taken from the duration of the iteration itself, and uses
// prometheus `rate` which has extrapolation of both sides of the bucket.
// Mem RSS is simply the Mem RSS value at the end of the iteration.
func (sd *DataHandle) OutputIteration(iterNumber int, f *os.File) {
	table := tablewriter.NewWriter(f)
	table.SetAutoFormatHeaders(false)
	table.SetHeader([]string{"Resource",
		fmt.Sprintf("CpuAvg (%ds)", int(sd.ItEndTime[iterNumber].Sub(sd.ItStartTime[iterNumber]).Seconds())),
		"Mem RSS"})

	sd.IterateTrackedPods(func(pod *corev1.Pod) {
		for _, cont := range pod.Spec.Containers {
			tableRow := []string{}

			resSeen := Resource{
				Namespace:     pod.Namespace,
				PodName:       pod.Name,
				ContainerName: cont.Name,
			}

			// Mark we saw this specific resource
			sd.SeenResources[resSeen] = true

			// shorten name
			resourceName := shortenResName(resSeen)
			tableRow = append(tableRow, resourceName)

			// CPU
			cpuIteration, err := sd.PromHandle.GetCPULoadAvgforContainer(pod.Namespace, pod.Name, cont.Name,
				sd.ItEndTime[iterNumber].Sub(sd.ItStartTime[iterNumber]),
				sd.ItEndTime[iterNumber])
			if err != nil {
				tableRow = append(tableRow, "-")
			} else {
				tableRow = append(tableRow, fmt.Sprintf("%.2f", cpuIteration))
			}

			// MemRSS
			memRSS, err := sd.PromHandle.GetMemRSSforContainer(pod.Namespace, pod.Name, cont.Name, sd.ItEndTime[iterNumber])
			if err != nil {
				tableRow = append(tableRow, "-")
			} else {
				tableRow = append(tableRow, humanize.Bytes(uint64(memRSS)))
			}
			table.Append(tableRow)
		}
	})
	table.Render()
}

// OutputIterationTable Print all iteration statistics in table format.
// For Duration and Memory values, relative distance to previous iteration is computed.
// For CPU, the CPU load average over the iteration time is computed (using prometheus rate).
func (sd *DataHandle) OutputIterationTable(f *os.File) {
	// Print all iteration information for all seen resources
	table := tablewriter.NewWriter(f)
	header := []string{"It", "Duration", "NPods"}
	rows := [][]string{}

	// Set up columns "It", "Duration", "NPods"
	var prevItDuration time.Duration
	for it := 0; it < sd.Iterations; it++ {
		itDurationString := ""

		// Duration of this iteration
		itDuration := sd.ItEndTime[it].Sub(sd.ItStartTime[it])
		itDurationString += itDuration.String()

		// Delta compared to previous iteration, if any
		if it > 0 {
			deltaString := percentDiff(float64(itDuration), float64(prevItDuration))
			itDurationString += fmt.Sprintf(" (%s)", deltaString)
		}
		prevItDuration = itDuration

		nPods, err := sd.PromHandle.GetNumEnvoysInMesh(sd.ItEndTime[it])
		var nPodsString string
		if err != nil {
			nPodsString = "err"
		} else {
			nPodsString = fmt.Sprintf("%d", nPods)
		}

		rows = append(rows, []string{
			fmt.Sprintf("%d", it),
			itDurationString,
			nPodsString,
		})
	}

	// Walk all seen tracked resources during test
	for seenRes := range sd.SeenResources {
		shortResName := shortenResName(seenRes)

		// append resname in header row
		header = append(header, shortResName)

		// Delta vars, to calculate deltas between iterations
		var previousMem float64
		for it := 0; it < sd.Iterations; it++ {
			// CPU
			var cpuString string
			cpuIt, err := sd.PromHandle.GetCPULoadAvgforContainer(
				seenRes.Namespace, seenRes.PodName, seenRes.ContainerName,
				sd.ItEndTime[it].Sub(sd.ItStartTime[it]), sd.ItEndTime[it])
			if err != nil {
				cpuString = errStr
			} else {
				cpuString = fmt.Sprintf("%.2f", cpuIt)
			}

			// Mem
			var memString string
			memIt, err := sd.PromHandle.GetMemRSSforContainer(
				seenRes.Namespace, seenRes.PodName, seenRes.ContainerName, sd.ItEndTime[it])
			if err != nil {
				memString = errStr
				previousMem = -1
			} else {
				memString += humanize.Bytes(uint64(memIt))

				// Check delta with previous iteration
				if it > 0 && previousMem != -1 {
					deltaString := percentDiff(float64(memIt), float64(previousMem))
					memString = fmt.Sprintf("%s (%s)", memString, deltaString)
				}
				previousMem = memIt
			}

			rows[it] = append(rows[it], fmt.Sprintf("%s / %s", cpuString, memString))
		}
	}
	table.SetAutoFormatHeaders(false)
	table.SetHeader(header)
	table.AppendBulk(rows)

	table.Render()
}

// OutputMemPerPod outputs the per-pod mem delta for the tracked resources over the test
// and outputs it in table format, example:
// The output requires at least 2 iterations to compare, as the MemRSS initial value is taken
// from the end of the first iteration, to avoid counting overhead of the application itself.
func (sd *DataHandle) OutputMemPerPod(f *os.File) {
	// Needs more than one iteration to do the diff at least
	if sd.Iterations <= 1 {
		return
	}

	table := tablewriter.NewWriter(f)
	table.SetAutoFormatHeaders(false)
	table.SetHeader([]string{"Resource", "Mem / pod"})

	for seenRes := range sd.SeenResources {
		row := []string{shortenResName(seenRes)}

		memRSSStart, err := sd.PromHandle.GetMemRSSforContainer(
			seenRes.Namespace, seenRes.PodName, seenRes.ContainerName, sd.ItEndTime[0])

		if err != nil && memRSSStart < 0 {
			table.Append(append(row, errStr))
			continue
		}

		memRSSEnd, err := sd.PromHandle.GetMemRSSforContainer(
			seenRes.Namespace, seenRes.PodName, seenRes.ContainerName, sd.ItEndTime[sd.Iterations-1])
		if err != nil && memRSSStart < 0 {
			table.Append(append(row, "err"))
			continue
		}

		numPodsStart, err := sd.PromHandle.GetNumEnvoysInMesh(sd.ItEndTime[0])
		if err != nil && memRSSStart < 0 {
			table.Append(append(row, "err"))
			continue
		}

		numPodsEnd, err := sd.PromHandle.GetNumEnvoysInMesh(sd.ItEndTime[sd.Iterations-1])
		if err != nil && memRSSStart < 0 {
			table.Append(append(row, "err"))
			continue
		}
		dv := uint64((memRSSEnd - memRSSStart) / float64(numPodsEnd-numPodsStart))
		table.Append(append(row, humanize.Bytes(dv)))
	}

	table.Render()
}

// OutputTestResults Print all available results
func (sd *DataHandle) OutputTestResults(f *os.File) {
	sd.OutputIterationTable(f)
	sd.OutputMemPerPod(f)
}
