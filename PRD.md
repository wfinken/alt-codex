Product Requirements Document (PRD): alt-codex

1. Executive Summary

alt-codex is a lightweight, terminal-based user interface (TUI) built using the Charm Bubbletea ecosystem in Go. It enables developers and power users to seamlessly manage, inspect, and switch between multiple Codex authentication profiles (e.g., separating work and personal accounts) without manual token wrangling or configuration edits. The tool provides an all-inclusive management suite covering profile switching, secure token storage, account addition/onboarding, and profile deletion directly from the terminal.

2. Goals & Objectives

Frictionless Switching: Reduce account switching to a sub-second keyboard operation inside the terminal.

Security First: Safely store authentication secrets using OS-native secure keychains (via zalando/go-keyring or similar) or encrypted local fallback stores.

Delightful Terminal UX: Leverage Bubbletea, Bubbles, and Lip Gloss to deliver a responsive, keyboard-driven, aesthetically pleasing TUI.

All-Inclusive Lifecycle Management: Handle the complete profile lifecycle—adding, renaming, switching, auditing, and removing accounts—within the application.

3. Target Audience & Use Cases

Target Audience: Software engineers, open-source contributors, and enterprise developers who juggle separate work and personal Codex profiles.

Key Use Cases:

Quick Switch: Launching alt-codex, viewing the active account, and jumping to another profile with a single keystroke.

Onboarding: Adding a new Codex account via an interactive login flow or manual token import.

Auditing: Verifying which profile is currently active and checking expiration timestamps or credential metadata.

4. User Experience & Interface Design (Charm Stack)

The interface will be built using Go and the Charm ecosystem:

Bubbletea (github.com/charmbracelet/bubbletea): The core Elm-architecture state management engine.

Bubbles (github.com/charmbracelet/bubbles): Reusable components such as text inputs (textinput), viewports, and spinners.

Lip Gloss (github.com/charmbracelet/lipgloss): Styling, layout, and color palettes optimized for modern terminal color schemes.

Screen Layouts / Views

Dashboard View (Main List):

Header banner displaying the alt-codex logo and current active profile indicator.

Scrollable list of profiles with status badges ([Active], [Saved], [Expired]).

Footer with contextual keybindings (a - Add, s - Switch, d - Delete, q - Quit).

Add Account View:

Form input fields for Profile Name (e.g., work-corp, personal-github) and Auth Token/Credentials.

Validation states and progress spinners during authentication verification.

Confirmation / Modal Dialogs:

Prompt dialogs for destructive actions like profile deletion or overwriting existing configurations.

5. Functional Requirements

5.1 Profile Management & Storage

FR-01: The app must support storing multiple named Codex profiles.

FR-02: Sensitive tokens and credentials must be stored securely using the OS keychain (macOS Keychain, Linux Secret Service, Windows Credential Manager) with a fallback to an encrypted local JSON config file (~/.config/alt-codex/config.enc) if keychains are unavailable.

FR-03: The configuration metadata (profile names, creation dates, active status) must be stored in ~/.config/alt-codex/profiles.json.

5.2 Account Switching

FR-04: Selecting a profile and pressing Enter or s must instantly update the active Codex CLI / application configuration files or environment state to point to the selected profile's credentials.

FR-05: The TUI must visually highlight the currently active profile with a distinct indicator (e.g., a green bullet point or checkmark).

5.3 Account Onboarding & Addition

FR-06: Users must be able to add a new account via an interactive form supporting a direct token paste or interactive login prompt.

FR-07: The app must validate the provided credentials before saving the profile to prevent storing malformed or dead tokens.

5.4 Profile Deletion & Cleanup

FR-08: Users must be able to delete profiles from the list, which will purge associated secrets from the OS keychain and remove metadata entries from configuration files.

FR-09: Deleting the currently active profile must trigger a warning prompt.

6. Non-Functional Requirements

NFR-01 (Performance): Startup time must be under 100ms, with profile switching completing instantaneously.

NFR-02 (Portability): Cross-platform support for macOS, Linux, and Windows terminals.

NFR-03 (Security): Zero plaintext storage of auth secrets on disk when OS keychains are accessible.

NFR-04 (Accessibility/Usability): Full keyboard navigation support (Arrow keys, Vim motions j/k, Esc, Enter) with clear visual feedback for all actions.

7. Future Scope & Roadmap (Phase 2)

Auto-refresh token support for profiles nearing expiration.

Shell integration helper (e.g., shell hooks to echo active profiles in prompt status bars).

Import/export utilities for backing up profile configurations securely.