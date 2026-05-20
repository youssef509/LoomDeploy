//go:build windows

package handlers

func getUptime() int64   { return 0 }
func getMemory() (int64, int64) { return 0, 0 }
func getDisk() (int64, int64)   { return 0, 0 }
func getCPU() float64    { return 0 }
