
package metrics

import (
	"context"
	"fmt"
	"time"

	// gopsutil provides cross-platform system monitoring APIs.
	// We use it to collect CPU, RAM, disk, and host information.
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
)

// CPUInfo contains overall CPU utilization and per-core statistics.
type CPUInfo struct {
	// Average CPU usage percentage across all logical CPUs.
	UsagePercent float64 `json:"usagePercent"`

	// Number of logical CPUs detected.
	Cores int `json:"cores"`

	// Individual usage percentage for each logical CPU.
	// For example: [15.2, 45.8, 28.4, 40.6]
	PerCoreUsage []float64 `json:"perCoreUsage"`
}

// MemoryInfo contains physical RAM utilization statistics.
// All memory sizes are reported in bytes.
type MemoryInfo struct {
	Total        uint64  `json:"total"`
	Used         uint64  `json:"used"`
	Available    uint64  `json:"available"`
	UsagePercent float64 `json:"usagePercent"`
}

// DiskInfo contains usage information for the root filesystem.
// Later, we can extend this to monitor multiple partitions.
type DiskInfo struct {
	Path         string  `json:"path"`
	Total        uint64  `json:"total"`
	Used         uint64  `json:"used"`
	Free         uint64  `json:"free"`
	UsagePercent float64 `json:"usagePercent"`
}

// HostInfo contains operating system and machine information.
type HostInfo struct {
	Hostname      string `json:"hostname"`
	OS            string `json:"os"`
	Platform      string `json:"platform"`
	KernelVersion string `json:"kernelVersion"`
	UptimeSeconds uint64 `json:"uptimeSeconds"`
}

// SystemMetrics combines all system information into one response.
// The JSON tags control the property names returned by our API.
type SystemMetrics struct {
	CPU       CPUInfo    `json:"cpu"`
	Memory    MemoryInfo `json:"memory"`
	Disk      DiskInfo   `json:"disk"`
	Host      HostInfo   `json:"host"`
	Timestamp string     `json:"timestamp"`
}

// CollectSystemMetrics gathers real-time Linux system metrics.
//
// The context allows the caller to cancel collection or enforce
// a timeout, preventing slow operations from blocking indefinitely.
//
// Returns:
//   - *SystemMetrics: collected system information
//   - error: details if metric collection fails
func CollectSystemMetrics(ctx context.Context) (*SystemMetrics, error) {

	// ---------------------------------------------------------
	// STEP 1: Collect CPU utilization for every logical CPU.
	// ---------------------------------------------------------

	// PercentWithContext measures CPU utilization over 1 second.
	//
	// Arguments:
	//   ctx         - cancellation and timeout context
	//   time.Second - measurement interval
	//   true        - return separate values for each logical CPU
	//
	// On a four-core system, the result might look like:
	// [15.2, 45.8, 28.4, 40.6]
	perCoreCPU, err := cpu.PercentWithContext(
		ctx,
		time.Second,
		true,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"collect per-core CPU metrics: %w",
			err,
		)
	}

	// Calculate overall CPU utilization by averaging
	// the individual logical CPU usage percentages.
	var cpuUsage float64

	for _, usage := range perCoreCPU {
		cpuUsage += usage
	}

	// Avoid division by zero if no CPU data was returned.
	if len(perCoreCPU) > 0 {
		cpuUsage /= float64(len(perCoreCPU))
	}

	// ---------------------------------------------------------
	// STEP 2: Collect physical RAM information.
	// ---------------------------------------------------------

	// VirtualMemoryWithContext retrieves RAM statistics.
	//
	// Total:     Total physical RAM.
	// Used:      RAM currently in use.
	// Available: RAM available for applications.
	// UsedPercent: Percentage of RAM currently used.
	memory, err := mem.VirtualMemoryWithContext(ctx)

	if err != nil {
		return nil, fmt.Errorf(
			"collect memory metrics: %w",
			err,
		)
	}

	// ---------------------------------------------------------
	// STEP 3: Collect root filesystem disk usage.
	// ---------------------------------------------------------

	// "/" represents the Linux root filesystem.
	//
	// This returns filesystem capacity, not physical disk
	// hardware details. We will add more partitions later.
	rootDisk, err := disk.UsageWithContext(ctx, "/")

	if err != nil {
		return nil, fmt.Errorf(
			"collect disk metrics: %w",
			err,
		)
	}

	// ---------------------------------------------------------
	// STEP 4: Collect host and operating system information.
	// ---------------------------------------------------------

	// Host information includes:
	// - Hostname
	// - Operating system
	// - Linux distribution
	// - Kernel version
	// - System uptime
	hostDetails, err := host.InfoWithContext(ctx)

	if err != nil {
		return nil, fmt.Errorf(
			"collect host information: %w",
			err,
		)
	}

	// ---------------------------------------------------------
	// STEP 5: Construct our API response.
	// ---------------------------------------------------------

	// Create a SystemMetrics object containing all
	// the information collected above.
	result := &SystemMetrics{

		// CPU statistics.
		CPU: CPUInfo{
			// Overall average utilization.
			UsagePercent: cpuUsage,

			// Number of logical CPUs detected.
			Cores: len(perCoreCPU),

			// Individual CPU utilization percentages.
			PerCoreUsage: perCoreCPU,
		},

		// Physical memory statistics.
		Memory: MemoryInfo{
			Total:        memory.Total,
			Used:         memory.Used,
			Available:    memory.Available,
			UsagePercent: memory.UsedPercent,
		},

		// Root filesystem disk statistics.
		Disk: DiskInfo{
			Path:         "/",
			Total:        rootDisk.Total,
			Used:         rootDisk.Used,
			Free:         rootDisk.Free,
			UsagePercent: rootDisk.UsedPercent,
		},

		// Operating system information.
		Host: HostInfo{
			Hostname:      hostDetails.Hostname,
			OS:            hostDetails.OS,
			Platform:      hostDetails.Platform,
			KernelVersion: hostDetails.KernelVersion,
			UptimeSeconds: hostDetails.Uptime,
		},

		// Timestamp indicating when this snapshot was collected.
		// RFC3339 produces a standard ISO 8601-compatible timestamp.
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	// Return the collected metrics to the HTTP handler.
	return result, nil
}
