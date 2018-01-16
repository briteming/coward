//  Crypto-Obscured Forwarder
//
//  Copyright (C) 2018 Rui NI <ranqus@gmail.com>
//
//  This file is part of Crypto-Obscured Forwarder.
//
//  Crypto-Obscured Forwarder is free software: you can redistribute it
//  and/or modify it under the terms of the GNU General Public License
//  as published by the Free Software Foundation, either version 3 of
//  the License, or (at your option) any later version.
//
//  Crypto-Obscured Forwarder is distributed in the hope that it will be
//  useful, but WITHOUT ANY WARRANTY; without even the implied warranty
//  of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//  GNU General Public License for more details.
//
//  You should have received a copy of the GNU General Public License
//  along with Crypto-Obscured Forwarder. If not, see
//  <http://www.gnu.org/licenses/>.

package config

import (
	"fmt"

	"github.com/reinit/coward/common/print"
)

type help struct {
	Fields fields
	Indent int
}

const helpItemDescriptionGap = 4
const helpItemStringFmtFirstLine = "%%%ds    %%s\r\n"

func (c *configurator) buildHelp(fs fields, parentIndent int) help {
	helpFields := make(fields, 0, 256)
	baseIndent := 0

	for _, field := range fs {
		helpFields = append(helpFields, field)

		tagLen := len(field.Tag)

		if tagLen > baseIndent {
			baseIndent = tagLen
		}
	}

	return help{
		Fields: helpFields,
		Indent: baseIndent + parentIndent,
	}
}

// Help print help information of current Configuration to a printer
func (c *configurator) Help(w print.Common) {
	helpItems := make([]help, 0, 256)

	helpItems = append(helpItems, c.buildHelp(c.fields, 4))

	for {
		if len(helpItems) <= 0 {
			break
		}

		helpCard := helpItems[0]

		helpItems = helpItems[1:]

		helpString := fmt.Sprintf(helpItemStringFmtFirstLine, helpCard.Indent)

		for fieldIdx, field := range helpCard.Fields {
			contentIndent := helpCard.Indent + helpItemDescriptionGap

			w.Writeln(
				[]byte(fmt.Sprintf(helpString, field.Tag, field.Description)),
				0, contentIndent, 1)

			if len(field.Sub) > 0 {
				helpItems = append([]help{c.buildHelp(
					field.Sub, contentIndent,
				)}, helpItems...)

				helpItems = append(helpItems, help{
					Fields: helpCard.Fields[fieldIdx+1:],
					Indent: helpCard.Indent,
				})

				break
			}
		}
	}
}
