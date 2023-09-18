package table

import (
	"github.com/charmbracelet/bubbles/table"
)

type Grid interface {
	Render() ([]table.Column, []table.Row)
}
