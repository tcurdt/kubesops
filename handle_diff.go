package main

import (
	"fmt"
	"sort"
)

// handleDiff compares secrets
// if path2 is empty: compare local (path1) vs remote
// if path2 is provided: compare local (path1) vs local (path2)
func handleDiff(path1, path2 string, verbose bool) error {
	if path2 == "" {
		// local vs remote comparison
		return upload(path1, false, false, verbose)
	}
	// local vs local comparison
	return diffLocalVsLocal(path1, path2, verbose)
}

// diffLocalVsLocal compares two local secrets
func diffLocalVsLocal(path1, path2 string, verbose bool) error {

	secrets1, err := LoadSecretsFromPath(path1)
	if err != nil {
		return fmt.Errorf("failed to load secrets from %s: %w", path1, err)
	}

	secrets2, err := LoadSecretsFromPath(path2)
	if err != nil {
		return fmt.Errorf("failed to load secrets from %s: %w", path2, err)
	}

	// single files (both have exactly 1 secret)
	if len(secrets1) == 1 && len(secrets2) == 1 {

		s1 := secrets1[0]
		s2 := secrets2[0]

		differences := compareSecretsLocal(s1.Namespace, s2.Namespace, s1.Data, s2.Data, verbose)
		fmt.Printf("%d difference(s)\n", differences)

		return nil
	}

	// multiple secrets - match by name only (ignore namespace)
	map1 := make(map[string]*Secret)
	map2 := make(map[string]*Secret)

	for _, s := range secrets1 {
		map1[s.Name] = s
	}

	for _, s := range secrets2 {
		map2[s.Name] = s
	}

	// get all keys
	allKeys := make(map[string]bool)
	for k := range map1 {
		allKeys[k] = true
	}
	for k := range map2 {
		allKeys[k] = true
	}

	// sort keys for consistent output
	keys := make([]string, 0, len(allKeys))
	for k := range allKeys {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	totalDifferences := 0

	for _, key := range keys {
		s1, exists1 := map1[key]
		s2, exists2 := map2[key]

		if !exists1 {
			fmt.Printf("secret %s: only in %s\n", key, path2)
			totalDifferences++
			continue
		}

		if !exists2 {
			fmt.Printf("secret %s: only in %s\n", key, path1)
			totalDifferences++
			continue
		}

		differences := compareSecretsLocal(s1.Namespace, s2.Namespace, s1.Data, s2.Data, verbose)
		totalDifferences += differences
	}

	fmt.Printf("%d difference(s)\n", totalDifferences)

	return nil
}

// truncateValue truncates a string to maxLen characters and adds "..." if truncated
func truncateValue(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// formatValue formats a value for display, with optional truncation
func formatValue(val string, exists bool, verbose bool) string {
	if !exists {
		return "missing"
	}
	if verbose {
		return "[" + val + "]"
	}
	return "[" + truncateValue(val, 10) + "]"
}

// compareSecretsLocal compares two local secrets and prints differences
// returns the number of differences found
func compareSecretsLocal(ns1, ns2 string, data1, data2 map[string]string, verbose bool) int {
	differences := 0

	// get all unique keys
	allKeys := make(map[string]bool)
	for k := range data1 {
		allKeys[k] = true
	}
	for k := range data2 {
		allKeys[k] = true
	}

	// sort keys for consistent output
	keys := make([]string, 0, len(allKeys))
	for k := range allKeys {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// show all keys, but only show values when they differ
	for _, key := range keys {
		val1, exists1 := data1[key]
		val2, exists2 := data2[key]

		// always print the key
		fmt.Printf("%s\n", key)

		// only show values if there's a difference
		if !exists1 || !exists2 || (val1 != val2) {
			differences++
			fmt.Printf("  %s: %s\n", ns1, formatValue(val1, exists1, verbose))
			fmt.Printf("  %s: %s\n", ns2, formatValue(val2, exists2, verbose))
		}
	}

	return differences
}

// compareSecretsRemote compares local vs remote secrets and prints differences
// returns the number of differences found
func compareSecretsRemote(secretName string, onsite, remote map[string]string, verbose bool) int {
	differences := 0

	// get all unique keys
	allKeys := make(map[string]bool)
	for k := range onsite {
		allKeys[k] = true
	}
	for k := range remote {
		allKeys[k] = true
	}

	// sort keys for consistent output
	keys := make([]string, 0, len(allKeys))
	for k := range allKeys {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// show all keys, but only show values when they differ
	for _, key := range keys {
		onsiteVal, onsiteExists := onsite[key]
		remoteVal, remoteExists := remote[key]

		changed := !onsiteExists || !remoteExists || (onsiteVal != remoteVal)

		if changed || verbose {
			fmt.Printf("%s/%s\n", secretName, key)
		}

		// only show values if there's a difference
		if changed {
			differences++
			fmt.Printf("  remote: %s\n", formatValue(remoteVal, remoteExists, verbose))
			fmt.Printf("  onsite: %s\n", formatValue(onsiteVal, onsiteExists, verbose))
		}
	}

	return differences
}
