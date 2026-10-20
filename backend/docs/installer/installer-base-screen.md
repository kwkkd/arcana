# BaseScreen Architecture Guide

> Practical guide for implementing new installer screens and migrating existing ones using the BaseScreen architecture.

## 🏗️ **Architecture Overview**

The `BaseScreen` provides a unified foundation for all installer form screens, encapsulating:

- **State Management**: `initialized`, `hasChanges`, `focusedIndex`, `showValues`
- **Form Handling**: `fields []FormField`, viewport management, auto-scrolling
- **Navigation**: Composite ScreenID support, GoBack patterns
- **Layout**: Responsive horizontal/vertical layouts
- **Lists**: Optional dropdown lists with delegates

### **Core Components**

```go
type BaseScreen struct {
    // Dependencies (injected)
    controller *controllers.StateController
    styles     *styles.Styles
    window     *window.Window

    // State
    initialized  bool
    hasChanges   bool
    focusedIndex int
    showValues   bool

    // Form data
    fields       []FormField
    fieldHeights []int

    // UI components
    viewport    viewport.Model
    formContent string

    // Handlers (must be implemented)
    handler     BaseScreenHandler
    listHandler BaseListHandler // optional
}
```

### **Required Interfaces**

```go
type BaseScreenHandler interface {
    BuildForm() tea.Cmd
    GetFormSummary() string
    GetFormTitle() string
    GetHelpContent() string
    HandleSave() error
    HandleReset()
    OnFieldChanged(fieldIndex int, oldValue, newValue string)
    GetFormFields() []FormField
    SetFormFields(fields []FormField)
}

type BaseListHandler interface { // Optional
    GetList() *list.Model
    OnListSelectionChanged(oldSelection, newSelection string)
    GetListHeight() int
}
```

## 🚀 **Creating New Screens**

### **1. Basic Form Screen**

```go
// example_form.go
type ExampleFormModel struct {
    *BaseScreen
    config *controllers.ExampleConfig
}

func NewExampleFormModel(
    controller *controllers.StateController,
    styles *styles.Styles,
    window *window.Window,
    args []string,
) *ExampleFormModel {
    m := &ExampleFormModel{
        config: controller.GetExampleConfig(),
    }

    m.BaseScreen = NewBaseScreen(controller, styles, window, args, m, nil)
    return m
}

// Required interface implementations
func (m *ExampleFormModel) BuildForm() {
    fields := []FormField{}

    // Text field
    apiKeyInput := textinput.New()
    apiKeyInput.Placeholder = "Enter API key"
    apiKeyInput.EchoMode = textinput.EchoPassword
    apiKeyInput.SetValue(m.config.APIKey)

    fields = append(fields, FormField{
        Key:         "api_key",
        Title:       "API Key",
        Description: "Your service API key",
        Required:    true,
        Masked:      true,
        Input:       apiKeyInput,
        Value:       apiKeyInput.Value(),
    })

    // Boolean field
    enabledInput := textinput.New()
    enabledInput.Placeholder = "true/false"
    enabledInput.ShowSuggestions = true
    enabledInput.SetSuggestions([]string{"true", "false"})
    enabledInput.SetValue(fmt.Sprintf("%t", m.config.Enabled))

    fields = append(fields, FormField{
        Key:         "enabled",
        Title:       "Enabled",
        Description: "Enable or disable service",
        Required:    false,
        Masked:      false,
        Input:       enabledInput,
        Value:       enabledInput.Value(),
    })

    m.SetFormFields(fields)
}

func (m *ExampleFormModel) GetFormTitle() string {
    return "Example Service Configuration"
}

func (m *ExampleFormModel) GetHelpContent() string {
    return "Configure your Example service settings here."
}

func (m *ExampleFormModel) HandleSave() error {
    fields := m.GetFormFields()
    for _, field := range fields {
        switch field.Key {
        case "api_key":
            m.config.APIKey = field.Input.Value()
        case "enabled":
            m.config.Enabled = field.Input.Value() == "true"
        }
    }

    if m.config.APIKey == "" {
        return fmt.Errorf("API key is required")
    }

    return m.GetController().UpdateExampleConfig(m.config)
}

func (m *ExampleFormModel) HandleReset() {
    m.config = m.GetController().GetExampleConfig()
    m.BuildForm()
}

func (m *ExampleFormModel) OnFieldChanged(fieldIndex int, oldValue, newValue string) {
    // Additional validation logic if needed
}

func (m *ExampleFormModel) GetFormFields() []FormField {
    return m.BaseScreen.fields
}

func (m *ExampleFormModel) SetFormFields(fields []FormField) {
    m.BaseScreen.fields = fields
}

// Update method with field input handling
func (m *ExampleFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "tab":
            // Handle tab completion for boolean fields
            return m.handleTabCompletion()
        default:
            // Handle field input
            if cmd := m.HandleFieldInput(msg); cmd != nil {
                return m, cmd
            }
        }
    }

    // Base screen handling
    return m.BaseScreen.Update(msg)
}
```

### **2. Screen with List Selection**

```go
// list_form.go
type ListFormModel struct {
    *BaseScreen
    config         *controllers.ListConfig
    selectionList  list.Model
    delegate       *ExampleDelegate
}

func NewListFormModel(...) *ListFormModel {
    m := &ListFormModel{
        config: controller.GetListConfig(),
    }

    m.initializeList()
    m.BaseScreen = NewBaseScreen(controller, styles, window, args, m, m) // Both handlers
    return m
}

func (m *ListFormModel) initializeList() {
    items := []list.Item{
        ExampleOption("Option 1"),
        ExampleOption("Option 2"),
    }

    m.delegate = &ExampleDelegate{
        style: m.GetStyles().FormLabel,
        width: MinMenuWidth - 6,
    }

    m.selectionList = list.New(items, m.delegate, MinMenuWidth-6, 3)
    m.selectionList.SetShowStatusBar(false)
    m.selectionList.SetFilteringEnabled(false)
    m.selectionList.SetShowHelp(false)
    m.selectionList.SetShowTitle(false)
}

// BaseListHandler implementation
func (m *ListFormModel) GetList() *list.Model {
    return &m.selectionList
}

func (m *ListFormModel) OnListSelectionChanged(oldSelection, newSelection string) {
    m.config.SelectedOption = newSelection
    m.BuildForm() // Rebuild form based on selection
}

func (m *ListFormModel) GetListHeight() int {
    return 5
}

func (m *ListFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        // Handle list input first
        if cmd := m.HandleListInput(msg); cmd != nil {
            return m, cmd
        }

        // Then field input
        if cmd := m.HandleFieldInput(msg); cmd != nil {
            return m, cmd
        }
    }

    return m.BaseScreen.Update(msg)
}
```

> The two examples above predate two changes to the base: `NewBaseScreen` no longer takes an `args []string` parameter, and `BuildForm()` now returns a `tea.Cmd`.

## 🔄 **Migrating Existing Screens**

### **Step 1: Analyze Current Screen**

From existing screen, identify:
- Form fields and their types
- List components (if any)
- Special keyboard handling
- Save/reset logic

### **Step 2: Refactor Structure**

```go
// Before:
type OldFormModel struct {
    controller   *controllers.StateController
    styles       *styles.Styles
    window       *window.Window
    args         []string
    initialized  bool
    hasChanges   bool
    focusedIndex int
    showValues   bool
    fields       []FormField
    viewport     viewport.Model
    formContent  string
    fieldHeights []int
    // ... config-specific fields
}

// After:
type NewFormModel struct {
    *BaseScreen                    // Embedded base screen
    // Only config-specific fields
    config *controllers.Config
    list   list.Model             // If needed
}
```

### **Step 3: Update Constructor**

```go
// Before:
func NewOldFormModel(...) *OldFormModel {
    return &OldFormModel{
        controller: controller,
        styles:     styles,
        // ... lots of boilerplate
    }
}

// After:
func NewNewFormModel(...) *NewFormModel {
    m := &NewFormModel{
        config: controller.GetConfig(),
    }

    // Initialize list if needed
    m.initializeList()

    m.BaseScreen = NewBaseScreen(controller, styles, window, m, m)
    return m
}
```

### **Step 4: Implement Required Interfaces**

Move existing methods to interface implementations:

```go
// Move: buildForm() → BuildForm()
// Move: save logic → HandleSave()
// Move: reset logic → HandleReset()
// Move: help content → GetHelpContent()
```

### **Step 5: Remove Redundant Methods**

Delete these methods from migrated screens:
- `getInputWidth()`, `getViewportSize()`, `updateViewport()`
- `focusNext()`, `focusPrev()`, `toggleShowValues()`
- `renderVerticalLayout()`, `renderHorizontalLayout()`
- `ensureFocusVisible()`

## 📋 **Environment Variable Integration**

Follow existing patterns for environment variable handling:

```go
func (m *FormModel) BuildForm() tea.Cmd {
    // Track initially set fields for cleanup
    m.initiallySetFields = make(map[string]bool)

    for _, fieldConfig := range m.fieldConfigs {
        envVar, _ := m.GetController().GetVar(fieldConfig.EnvVarName)
        m.initiallySetFields[fieldConfig.Key] = envVar.IsPresent()

        field := m.createFieldFromEnvVar(fieldConfig, envVar)
        fields = append(fields, field)
    }

    m.SetFormFields(fields)

    return nil // a form screen has nothing to start; only a panel screen returns a cmd
}

func (m *FormModel) HandleSave() error {
    // First pass: Remove cleared fields
    for _, field := range m.GetFormFields() {
        value := strings.TrimSpace(field.Input.Value())
        if value == "" && m.initiallySetFields[field.Key] {
            m.GetController().SetVar(field.EnvVarName, "")
        }
    }

    // Second pass: Save non-empty values
    for _, field := range m.GetFormFields() {
        value := strings.TrimSpace(field.Input.Value())
        if value != "" {
            m.GetController().SetVar(field.EnvVarName, value)
        }
    }

    return nil
}
```

## 🎯 **Navigation Integration**

### **Screen Registration — four mandatory steps**

All four are required. Skipping any of them fails *quietly*: nothing panics and nothing is logged. Three of them show a symptom somewhere other than the code you changed; the fourth — step 2 — shows no symptom at all with the screens written so far, which is exactly why it is the one that gets skipped.

**1. The `ScreenID` constant** in `wizard/models/types.go`:

```go
const (
    ExampleFormScreen ScreenID = "example_form"
    // parameterised screens join their arguments with "§"
    UpdatePentagiScreen ScreenID = "processor_operation_form§compose§update"
)
```

The ID is the only key the navigator, the registry and every menu share. Without it there is nothing to push onto the navigation stack and nothing to key the registry by.

**2. A case in `models.RestoreModel`** (`types.go`):

```go
case *ExampleFormModel:
    return m
```

`app.forwardMsgToCurrentModel` calls `Update` on the current model and adopts the result only when `RestoreModel` recognises it — when it returns `nil` there is no fallback and no log: the app keeps the pointer it already holds, and returns the command either way. What a missing branch actually costs is worth stating precisely, because the obvious answer is wrong. Every screen in `wizard/models` has a pointer receiver on `Update` and returns *itself*, so the object the app renders is the object `Update` has just mutated — nothing freezes, and a missing branch shows no symptom at all. The branch is load-bearing for the case the type allows but nobody has written yet: a model that returns a *different* model, dropped silently with nothing anywhere to say why. That is why the contract is "every screen is listed" rather than "list the ones that need it" — the day a screen starts returning something else is not the day anybody remembers this rule. The `nil` is load-bearing in the other direction too: a `MaintenanceModel` delegates `Update` to its embedded `*ListScreen`, which returns *itself*, and `RestoreModel` not knowing `*ListScreen` is exactly what keeps the app pointing at the whole screen instead of one of its parts — so list the concrete screen type and nothing else.

**3. Construction in `registry.initScreens`** (`wizard/registry/registry.go`):

```go
r.screens[models.ExampleFormScreen] = models.NewExampleFormModel(r.controller, r.styles, r.window)
```

Screens are built once here and reused; `registry.GetScreen` falls back to a `MockFormModel` for an unknown ID **and caches it under that ID**, so the screen you wrote is never constructed at all. The first symptom is not the screen but the menu: `ListScreen` labels each entry by asking the registry for the model behind its ID (`GetFormName`, `GetFormOverview`, `GetCurrentConfiguration`, `IsConfigured`), so the entry reads "Development Screen" and opens a placeholder. An unregistered screen also never receives `tea.WindowSizeMsg`, which `registry.HandleMsg` forwards to registered screens only.

**4. Strings in `wizard/locale/locale.go`, and hotkey captions in `app.initHotkeysLocale`** (`wizard/app.go`). All user-visible text lives in `locale`; the footer resolves each name from `GetFormHotKeys()` through the caption map:

```go
if localeHotKey, ok := app.hotkeys[hotkey]; ok {
    actions = append(actions, localeHotKey)
}
```

An unknown name is skipped without complaint, so the key keeps working and is simply never announced — the quietest of the four failures, and the only one whose symptom is a hint that is missing rather than a screen that is wrong. `enter`, `up|down`, `pgup|pgdown`, `home|end` and `y|n` are already in the map.

### **Width-aware markdown rendering**

`styles.GetRenderer()` returns a single renderer built once with `glamour.WithWordWrap(DefaultRenderWidth)` — 80 columns, whatever the terminal is doing. That was enough while the only markdown screen was the EULA, which renders once at load and is prose written to be read at a fixed measure. It is not enough for a document that fills the screen: at 80 columns in a 200-column window the text uses a third of the width, and in a terminal narrower than 80 every line breaks two or three times.

`styles.GetRendererForWidth(width)` is the replacement for screens that know how much room they have:

```go
func (s *Styles) GetRendererForWidth(width int) *glamour.TermRenderer {
    width = min(max(width, MinRenderWidth), MaxRenderWidth) // [40, 120]
    // ... memoised per width; on any failure returns the fixed 80-column renderer
}
```

Four properties are worth knowing before you use it:

- **It clamps to `[MinRenderWidth, MaxRenderWidth]` = `[40, 120]`.** The floor keeps a narrow terminal from splitting every line; the ceiling is a readability limit — a line spanning a maximised terminal is measurably harder to read.
- **It never returns `nil`.** A width that cannot be built falls back to the default renderer, so no render path needs a nil check — and a nil check that someone forgets is how a document ends up not being shown at all.
- **The cache is behind a pointer** (`renderers *rendererCache`), because `styles.Styles` is copied by value into every screen and a value cache would be cloned per screen, defeating the point.
- **The cache is mutex-guarded**, because renderers are built inside `tea.Cmd` goroutines — off the update loop, where two screens can ask for the same width at once.

`GetRenderer()` still exists and still wraps at 80; prefer `GetRendererForWidth` on anything that renders inside a sized panel.

### **Navigation Usage**

```go
// Navigate to screen with parameters
return m, func() tea.Msg {
    return NavigationMsg{
        Target: CreateScreenID("example_form", "config_type"),
    }
}

// Return to previous screen
return m, func() tea.Msg {
    return NavigationMsg{GoBack: true}
}
```

## 🔧 **Interface Validation**

Add compile-time interface checks:

```go
// Ensure interfaces are implemented
var _ BaseScreenHandler = (*ExampleFormModel)(nil)
var _ BaseListHandler = (*ListFormModel)(nil)

// A screen that embeds BaseScreen asserts both: the registry stores it as a model,
// and BaseScreen calls back into it as a handler
var _ BaseScreenModel = (*ApplyChangesFormModel)(nil)
var _ BaseScreenHandler = (*ApplyChangesFormModel)(nil)

// A document screen has no form, so it asserts only the model interface
var _ BaseScreenModel = (*EULAModel)(nil)
```

## 📊 **Benefits Summary**

- **Code Reduction**: 50-60% less boilerplate per screen
- **Consistency**: Unified behavior across all forms
- **Maintainability**: Centralized bug fixes and improvements
- **Development Speed**: Faster new screen implementation

## 🎯 **Quick Reference**

### **Screen Types**

1. **Simple Form**: Inherit BaseScreen, implement BaseScreenHandler
2. **Form with List**: Inherit BaseScreen, implement both handlers
3. **Menu Screen**: Use existing patterns without BaseScreen
4. **Document**: No BaseScreen — implement BaseScreenModel over a viewport (`eula.go`)
5. **Action with terminal panel**: Inherit BaseScreen with zero fields, own the terminal and override `View()` (`apply_changes.go`)

### **Required Methods**

- `BuildForm()` - Create form fields
- `HandleSave()` - Save configuration with validation
- `HandleReset()` - Reset to defaults
- `GetFormTitle()` - Screen title
- `GetHelpContent()` - Right panel content

### **Optional Methods**

- `OnFieldChanged()` - Real-time validation
- List handler methods (if using lists)

This architecture enables rapid development of new installer screens while maintaining consistency and reducing code duplication.
