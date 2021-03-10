package deal

import (
	"context"
	"fmt"
	"io"
	"time"

	tm "github.com/buger/goterm"
	"github.com/fatih/color"
	lapi "github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/lib/tablewriter"
	term "github.com/nsf/termbox-go"
)

var mockDealInfos []lapi.DealInfo

func ShowDealsCmd(ctx context.Context, api lapi.FullNode) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	localDeals, err := api.ClientListDeals(ctx)
	if err != nil {
		return err
	}

	return showDealsUX(ctx, localDeals)
}

func showDealsUX(ctx context.Context, deals []lapi.DealInfo) error {
	err := term.Init()
	if err != nil {
		return err
	}
	defer term.Close()

	renderer := dealRenderer{out: tm.Screen}

	dealIdx := -1
	state := "main"
	highlighted := -1
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		switch state {
		case "main":
			renderMain := func(hlite int) error {
				tm.Clear()
				tm.MoveCursor(1, 1)
				err := renderer.renderDeals(deals, hlite)
				if err != nil {
					return err
				}
				tm.Flush()
				return nil
			}
			err := renderMain(highlighted)
			if err != nil {
				return err
			}

			switch ev := term.PollEvent(); ev.Type {
			case term.EventKey:
				switch {
				case ev.Ch == 'q', ev.Key == term.KeyEsc:
					return nil
				case ev.Key == term.KeyArrowUp:
					term.Sync()
					if highlighted > 0 {
						highlighted--
					}
				case ev.Key == term.KeyArrowDown:
					term.Sync()
					highlighted++
				case ev.Key == term.KeyEnter:
					term.Sync()
					dealIdx = highlighted
					state = "deal"
				}
			case term.EventError:
				return ev.Err
			}
		case "deal":
			tm.Clear()
			tm.MoveCursor(1, 1)
			renderer.renderDeal(deals[dealIdx])
			tm.Flush()

			switch ev := term.PollEvent(); ev.Type {
			case term.EventKey:
				if ev.Ch == 'q' || ev.Key == term.KeyEsc || ev.Key == term.KeyEnter || ev.Key == term.KeyArrowLeft {
					term.Sync()
					state = "main"
				}
			case term.EventError:
				return ev.Err
			}
		}
	}
}

type dealRenderer struct {
	out io.Writer
}

func (r *dealRenderer) renderDeals(deals []lapi.DealInfo, highlighted int) error {
	tw := tablewriter.New(
		tablewriter.Col(""),
		tablewriter.Col("Created"),
		tablewriter.Col("Provider"),
		tablewriter.Col("Size"),
		tablewriter.Col("State"),
	)
	for i, di := range deals {
		lineNum := fmt.Sprintf("%d", i+1)
		cols := map[string]interface{}{
			"":         lineNum,
			"Created":  time.Since(di.CreationTime).Round(time.Second),
			"Provider": di.Provider,
			"Size":     di.Size,
			//"State":    di.Message,
		}
		if i == highlighted {
			for k, v := range cols {
				cols[k] = color.YellowString(fmt.Sprint(v))
			}
		}
		tw.Write(cols)
	}
	return tw.Flush(r.out)
}

func (r *dealRenderer) renderDeal(di lapi.DealInfo) error {
	_, err := fmt.Fprintf(r.out, "Deal %d\n", di.DealID)
	if err != nil {
		return err
	}
	for _, stg := range di.DealStages.Stages {
		msg := fmt.Sprintf("%s: %s (%s)", stg.Name, stg.Description, stg.ExpectedDuration)
		if stg.UpdatedTime.Time().IsZero() {
			msg = color.YellowString(msg)
		}
		_, err := fmt.Fprintf(r.out, "%s\n", msg)
		if err != nil {
			return err
		}
		for _, l := range stg.Logs {
			_, err = fmt.Fprintf(r.out, "  %s %s\n", time.Since(l.UpdatedTime.Time()).Round(time.Second), l.Log)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
