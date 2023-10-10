package scraper

import "context"

type job struct {
	target   scrapeTarget
	callback func()
}

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
