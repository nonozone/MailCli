package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/yourname/mailcli/pkg/schema"
	"gopkg.in/yaml.v3"
)

func writeMessage(out io.Writer, msg *schema.StandardMessage, format string) error {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "json":
		return writeJSON(out, msg)
	case "yaml":
		return writeYAML(out, msg)
	case "table":
		return writeTable(out, msg)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

func writeJSON(out io.Writer, msg *schema.StandardMessage) error {
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(msg)
}

func writeYAML(out io.Writer, msg *schema.StandardMessage) error {
	data, err := yaml.Marshal(msg)
	if err != nil {
		return err
	}

	if _, err := out.Write(data); err != nil {
		return err
	}
	_, err = out.Write([]byte("\n"))
	return err
}

func writeTable(out io.Writer, msg *schema.StandardMessage) error {
	table := tablewriter.NewWriter(out)
	table.SetHeader([]string{"Field", "Value"})
	table.AppendBulk([][]string{
		{"ID", msg.ID},
		{"From", formatAddress(msg.Meta.From)},
		{"To", formatAddresses(msg.Meta.To)},
		{"Subject", msg.Meta.Subject},
		{"Date", msg.Meta.Date},
		{"Format", msg.Content.Format},
		{"Snippet", msg.Content.Snippet},
		{"Category", msg.Content.Category},
		{"Actions", formatActions(msg.Actions)},
		{"Labels", strings.Join(msg.Labels, ", ")},
	})
	table.Render()
	return nil
}

func formatAddress(addr *schema.Address) string {
	if addr == nil {
		return ""
	}
	if addr.Name == "" {
		return addr.Address
	}
	return fmt.Sprintf("%s <%s>", addr.Name, addr.Address)
}

func formatAddresses(addrs []schema.Address) string {
	if len(addrs) == 0 {
		return ""
	}
	values := make([]string, 0, len(addrs))
	for _, addr := range addrs {
		addrCopy := addr
		values = append(values, formatAddress(&addrCopy))
	}
	return strings.Join(values, ", ")
}

func formatActions(actions []schema.Action) string {
	if len(actions) == 0 {
		return ""
	}
	values := make([]string, 0, len(actions))
	for _, action := range actions {
		if action.URL != "" {
			values = append(values, fmt.Sprintf("%s: %s", action.Type, action.URL))
			continue
		}
		values = append(values, action.Type)
	}
	return strings.Join(values, ", ")
}
