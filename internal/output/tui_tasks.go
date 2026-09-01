package output

import (
	"fmt"
	"slices"

	tea "charm.land/bubbletea/v2"
)

func (m *tuiModel) scheduleTask(invocation TaskInvocation) *tuiTask {
	if task := m.byID[invocation.ID]; task != nil {
		return task
	}
	isRoot := invocation.ID == invocation.RootID
	key := tuiTaskKey{rootID: invocation.RootID, name: invocation.Name, isRoot: isRoot}
	m.nameCounts[key]++
	task := &tuiTask{
		id:           invocation.ID,
		rootID:       invocation.RootID,
		name:         invocation.Name,
		occurrence:   m.nameCounts[key],
		internal:     invocation.Internal,
		isRoot:       isRoot,
		hidden:       m.hideInternal && invocation.Internal && !isRoot,
		state:        taskPending,
		followOutput: true,
	}
	m.byID[invocation.ID] = task
	m.tasks = append(m.tasks, task)
	if m.hasSelect && m.selectedID == task.id {
		m.loadViewport()
	}
	if !task.isRoot && !task.hidden && !m.hasSelect {
		m.selectedID = task.id
		m.hasSelect = true
		m.loadViewport()
	}
	return task
}

func (m *tuiModel) joinTask(id, ownerID uint64) {
	task := m.byID[id]
	if task == nil {
		return
	}
	selected := m.hasSelect && m.selectedID == id
	if selected {
		m.saveViewport()
	}
	delete(m.byID, id)
	for i, candidate := range m.tasks {
		if candidate.id == id {
			m.tasks = append(m.tasks[:i], m.tasks[i+1:]...)
			break
		}
	}
	m.recountTaskNames(tuiTaskKey{rootID: task.rootID, name: task.name, isRoot: task.isRoot})
	if selected {
		m.selectedID = ownerID
		m.hasSelect = ownerID != 0
		m.loadViewport()
	}
	m.keepSelectionVisible()
}

func (m *tuiModel) recountTaskNames(key tuiTaskKey) {
	count := 0
	for _, task := range m.tasks {
		if (tuiTaskKey{rootID: task.rootID, name: task.name, isRoot: task.isRoot}) != key {
			continue
		}
		count++
		task.occurrence = count
	}
	if count == 0 {
		delete(m.nameCounts, key)
		return
	}
	m.nameCounts[key] = count
}

func (m *tuiModel) ensureOutputTask(id uint64, name string) *tuiTask {
	if task := m.byID[id]; task != nil {
		return task
	}
	if name == "" {
		name = fmt.Sprintf("task %d", id)
	}
	task := &tuiTask{id: id, name: name, state: taskPending, followOutput: true}
	m.byID[id] = task
	m.tasks = append(m.tasks, task)
	if !m.hasSelect {
		m.selectedID = task.id
		m.hasSelect = true
		m.loadViewport()
	}
	return task
}

func (m *tuiModel) appendOutput(id uint64, name, data string) {
	task := m.ensureOutputTask(id, name)
	if task.state == taskPending && id != 0 {
		task.state = taskRunning
	}
	task.output += normalizeOutput(data)
	if len(task.output) > maxTaskOutputLen {
		task.output = task.output[len(task.output)-maxTaskOutputLen:]
		task.truncated = true
	}
	if m.hasSelect && task.id == m.selectedID {
		m.loadViewport()
	}
}

func (m tuiModel) taskName(task *tuiTask) string {
	key := tuiTaskKey{rootID: task.rootID, name: task.name, isRoot: task.isRoot}
	if m.nameCounts[key] > 1 {
		return fmt.Sprintf("#%d %s", task.occurrence, task.name)
	}
	return task.name
}

type tuiTaskRow struct {
	task       *tuiTask
	treePrefix string
}

func (m tuiModel) taskRows() []tuiTaskRow {
	childrenByRoot := make(map[uint64][]*tuiTask)
	var roots, standalone []*tuiTask
	for _, task := range m.tasks {
		if task.hidden {
			continue
		}
		if task.isRoot {
			roots = append(roots, task)
			continue
		}
		if task.rootID == 0 {
			standalone = append(standalone, task)
		} else {
			childrenByRoot[task.rootID] = append(childrenByRoot[task.rootID], task)
		}
	}

	rows := make([]tuiTaskRow, 0, len(m.tasks))
	for _, root := range roots {
		rows = append(rows, tuiTaskRow{task: root})
		children := childrenByRoot[root.id]
		for i, child := range children {
			prefix := "├─ "
			if i == len(children)-1 {
				prefix = "└─ "
			}
			rows = append(rows, tuiTaskRow{task: child, treePrefix: prefix})
		}
	}
	for _, task := range standalone {
		rows = append(rows, tuiTaskRow{task: task})
	}
	return rows
}

func (m *tuiModel) selectedTask() *tuiTask {
	if !m.hasSelect {
		return nil
	}
	return m.byID[m.selectedID]
}

func (m *tuiModel) selectedIndex() int {
	for i, row := range m.taskRows() {
		if row.task.id == m.selectedID {
			return i
		}
	}
	return -1
}

func (m *tuiModel) moveSelection(delta int) {
	rows := m.taskRows()
	if len(rows) == 0 {
		return
	}
	index := m.selectedIndex()
	if index < 0 {
		if delta < 0 {
			m.selectBoundary(true)
		} else {
			m.selectBoundary(false)
		}
		return
	}
	for index += delta; index >= 0 && index < len(rows); index += delta {
		if !rows[index].task.isRoot {
			m.selectTask(index)
			return
		}
	}
}

func (m *tuiModel) selectTask(index int) {
	rows := m.taskRows()
	if index < 0 || index >= len(rows) || rows[index].task.isRoot {
		return
	}
	m.saveViewport()
	m.selectedID = rows[index].task.id
	m.hasSelect = true
	m.keepSelectionVisible()
	m.loadViewport()
}

func (m *tuiModel) selectBoundary(last bool) {
	rows := m.taskRows()
	if last {
		for i := range slices.Backward(rows) {
			if !rows[i].task.isRoot {
				m.selectTask(i)
				return
			}
		}
		return
	}
	for i, row := range rows {
		if !row.task.isRoot {
			m.selectTask(i)
			return
		}
	}
}

func (m *tuiModel) keepSelectionVisible() {
	index := m.selectedIndex()
	if index < 0 {
		return
	}
	visible := max(newTUILayout(m.width, m.height).innerHeight-1, 1)
	if index < m.listTop {
		m.listTop = index
	} else if index >= m.listTop+visible {
		m.listTop = index - visible + 1
	}
	maxTop := max(len(m.taskRows())-visible, 0)
	m.listTop = min(max(m.listTop, 0), maxTop)
}

func (m *tuiModel) resizeViewport() {
	layout := newTUILayout(m.width, m.height)
	m.viewport.SetWidth(layout.rightInnerWidth)
	m.viewport.SetHeight(max(layout.innerHeight-1, 1))
}

func (m *tuiModel) loadViewport() {
	task := m.selectedTask()
	if task == nil {
		m.viewport.SetContent("")
		return
	}
	content := task.output
	if task.truncated {
		content = tuiHelpStyle.Render("… earlier output was discarded …") + "\n" + content
	}
	m.viewport.SetContent(content)
	if task.followOutput {
		m.viewport.GotoBottom()
	} else {
		m.viewport.SetYOffset(task.scrollOffset)
	}
}

func (m *tuiModel) saveViewport() {
	task := m.selectedTask()
	if task == nil {
		return
	}
	task.scrollOffset = m.viewport.YOffset()
	task.followOutput = m.viewport.AtBottom()
}

func (m *tuiModel) updateViewport(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	m.saveViewport()
	return cmd
}

func (m *tuiModel) toggleFocus() {
	if m.focus == taskPane {
		m.focus = outputPane
	} else {
		m.focus = taskPane
	}
}

func (m *tuiModel) handleMouseClick(mouse tea.Mouse) {
	layout := newTUILayout(m.width, m.height)
	if mouse.Y < 0 || mouse.Y >= layout.bodyHeight {
		return
	}
	if mouse.X >= 0 && mouse.X < layout.leftOuterWidth {
		m.focus = taskPane
		// Border is row 0 and the title is row 1, so tasks begin at row 2.
		row := mouse.Y - 2
		if row >= 0 {
			m.selectTask(m.listTop + row)
		}
		return
	}
	if mouse.X >= layout.leftOuterWidth+layout.gap {
		m.focus = outputPane
	}
}

func (m *tuiModel) handleMouseWheel(msg tea.MouseWheelMsg) tea.Cmd {
	layout := newTUILayout(m.width, m.height)
	if msg.Y < 0 || msg.Y >= layout.bodyHeight {
		return nil
	}
	if msg.X < layout.leftOuterWidth {
		m.focus = taskPane
		switch msg.Button {
		case tea.MouseWheelUp:
			m.moveSelection(-1)
		case tea.MouseWheelDown:
			m.moveSelection(1)
		}
		return nil
	}
	if msg.X >= layout.leftOuterWidth+layout.gap {
		m.focus = outputPane
		return m.updateViewport(msg)
	}
	return nil
}
