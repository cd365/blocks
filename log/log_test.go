package log

import (
	"fmt"
	"github.com/rs/zerolog"
	"testing"
	"time"
)

func TestNewLogger(t *testing.T) {
	l := NewLogger(nil)

	fmt.Println(l.GetLevel().String())
	l.SetLevel(zerolog.DebugLevel)
	fmt.Println(l.GetLevel().String())

	l.CustomContext(func(ctx zerolog.Context) zerolog.Logger {
		return ctx.
			Str("project", "project-a").
			Str("module", "module-a").
			Int64("program_start_at", time.Now().Unix()).
			Logger()
	})

	// Note the difference in the `unix_milli` values in the two log outputs
	l.CustomEvent(func(event *zerolog.Event) *zerolog.Event {
		return event.Int64("unix_milli", time.Now().UnixMilli())
	})

	l.Info().Msg("123")

	<-time.After(time.Second * 2)

	l.Error().Err(fmt.Errorf("321")).Send()

	l.Trace().Msg("000")
}
