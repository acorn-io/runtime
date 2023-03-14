package table

import (
	"bytes"
	"strings"
)

func SimpleFormat(values [][]string) (string, string) {
	headerBuffer := bytes.Buffer{}
	valueBuffer := bytes.Buffer{}
	for _, v := range values {
		// add Column label in all caps
		appendTabDelim(&headerBuffer, strings.ToUpper(v[0]))
		if strings.Contains(v[1], "{{") {
			// Special formatting
			appendTabDelim(&valueBuffer, v[1])
		} else {
			// Default formatting, loop up name
			appendTabDelim(&valueBuffer, "{{."+v[1]+"}}")
		}
	}

	headerBuffer.WriteString("\n")
	valueBuffer.WriteString("\n")

	return headerBuffer.String(), valueBuffer.String()
}

func appendTabDelim(buf *bytes.Buffer, value string) {
	if buf.Len() == 0 {
		buf.WriteString(value)
	} else {
		buf.WriteString("\t")
		buf.WriteString(value)
	}
}
