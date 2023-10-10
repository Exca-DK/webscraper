package analytics

type Analyzer interface {
	Analyze(src string, words []string)
}

type FrequencyAnalyzer struct {
}

func (analyzer *FrequencyAnalyzer) Analyze(src string, _ []string) {
	// TODO finish analyzer
}
