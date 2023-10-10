package scraper

import "context"

// job represents a web scraping task with a target and a callback function.
// It encapsulates the information required for a single scraping operation,
// including the scrape target and a callback to be executed upon completion.
type job struct {
	target   scrapeTarget // The target for the scraping task.
	callback func()       // A callback function to execute after task completion.
}

// taskLoop is responsible for managing web scraping tasks within a worker thread.
// It continuously waits for and processes scraping jobs by fetching and analyzing web pages.
func (s *Scrapper) taskLoop(ctx context.Context, _ string) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case j := <-s.jobCh:
			currentIndex := s.jobIndex.Add(1) - 1
			err := s.scrape(ctx, currentIndex, j.target)
			if err != nil {
				// TODO log when added logging package
			}
			j.callback()
		}
	}
}
