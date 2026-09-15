// Package pipeline wires collector → detectors → correlate → classify → score →
// policy → reporters. Stages do not import each other; this package orchestrates.
package pipeline
