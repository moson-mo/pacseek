package pacseek

import "time"

// starts the spinner
func (ps *UI) startSpin() {
	chars := "⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏"
	if ps.asciiMode {
		chars = "|/-\\"
	}
	go func() {
		for {
			select {
			case <-ps.quitSpin:
				return
			default:
				ms := time.Duration(40)
				for _, c := range chars {
					ps.app.QueueUpdateDraw(func() {
						ps.spinner.SetText(string(c))
					})
					time.Sleep(ms * time.Millisecond)
				}
			}
		}
	}()
}

// stops the spinner
func (ps *UI) stopSpin() {
	ps.quitSpin <- true
}
