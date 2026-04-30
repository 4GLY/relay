import type { Dictionary } from "../types";

export const en: Dictionary = {
  common: {
    continueWithGitHub: "Continue with GitHub",
    unknownError: "Something went wrong.",
    links: {
      backToOnboarding: "Back to onboarding",
      projectExplorer: "Project Explorer",
      providerSettings: "Provider settings",
      apiKeys: "API key settings",
      styleMemory: "Style Memory",
    },
    language: {
      label: "Language",
      english: "English",
      korean: "Korean",
    },
  },
  root: {
    eyebrow: "4gly Labs · Relay",
    title: "Relay",
    subtitle: "A quiet engine that turns chaos into swans.",
    panelTitle: "Sign in to start",
    panelCopy:
      "Relay creates a private workspace first. Provider keys stay in Settings, not in first-run setup.",
    signInButton: "Continue with GitHub",
  },
  onboarding: {
    page: {
      eyebrow: "Slice 8 · 60 seconds",
      title: "First run, no keys",
      subtitle:
        "Relay starts by creating a private workspace. Provider keys are a Settings concern, not a gate on the first minute.",
      signInTitle: "Sign in to create your workspace",
      signInCopy:
        "Use an identity provider first. Relay will create your Personal project after you return here.",
    },
    client: {
      signedInEyebrowPrefix: "Signed in",
      fallbackUser: "Relay user",
      title: "Create your Relay workspace",
      copy:
        "Relay will create your Personal project and send you straight into Project Explorer. Claude provider keys stay out of first-run setup.",
      startButton: "Start in Relay",
      startingButton: "Creating workspace...",
      styleMemoryLink: "Skip to Style Memory",
      providerSettingsLink: "Provider settings",
      apiKeysLink: "API key settings",
      fallbackError: "Could not finish onboarding.",
    },
  },
  providers: {
    page: {
      eyebrow: "Settings · provider credentials",
      signInTitle: "Sign in first",
      signInCopy: "Provider credentials are user-owned settings.",
      loadErrorTitle: "Could not load provider settings",
      loadErrorCopy: "Try refreshing this page after the Relay API is reachable again.",
    },
    client: {
      eyebrow: "Settings · provider credentials",
      title: "Claude provider",
      copy:
        "Connect Anthropic only when Claude-backed features need it. This key is not part of first-run onboarding.",
      settingsOnlyPill: "Settings only · optional after onboarding",
      connected: "Connected",
      disconnected: "Not connected",
      noStoredKey: "No provider key is stored for this user.",
      maskedKeySeparator: "ending",
      savingStatus: "Encrypting and saving provider key.",
      disconnectingStatus: "Removing stored provider key.",
      connectedHelp:
        "Settings only. Your key is available to Claude-backed features but first-run onboarding stays keyless.",
      disconnectedHelp:
        "Settings only. Add a key later when Claude-backed features need one.",
      apiKeyLabel: "Anthropic API key",
      apiKeyPlaceholder: "sk-ant-...",
      fieldHelp:
        "The raw key is encrypted before storage. Relay returns only masked metadata here.",
      connectButton: "Connect key",
      replaceButton: "Replace key",
      savingButton: "Saving...",
      disconnectButton: "Disconnect",
      disconnectingButton: "Disconnecting...",
      fallbackConnectError: "Could not connect provider.",
      fallbackDisconnectError: "Could not disconnect provider.",
    },
    errorMap: {
      UNAUTHENTICATED: "Sign in again to manage provider credentials.",
      "Anthropic keys must start with sk-ant-": "Anthropic keys must start with sk-ant-",
    },
  },
  apiKeys: {
    page: {
      eyebrow: "Settings · Relay API keys",
      signInTitle: "Sign in first",
      signInCopy: "Relay API keys are user-owned settings for API and MCP access.",
      loadErrorTitle: "Could not load API keys",
      loadErrorCopy: "Try refreshing this page after the Relay API is reachable again.",
    },
    client: {
      eyebrow: "Settings · Relay API keys",
      title: "Relay API keys",
      copy:
        "Issue keys here for external agents and tools that need Relay API or MCP access. Keep provider credentials separate.",
      settingsOnlyPill: "Settings only · user-owned access tokens",
      nameLabel: "Key name",
      namePlaceholder: "CLI on laptop",
      fieldHelp: "Relay shows the raw token only once. Copy it now and store it safely.",
      issueButton: "Issue key",
      issuingButton: "Issuing...",
      issuedSuccess: "Issued a new Relay API key.",
      revokedSuccess: "Revoked the Relay API key.",
      tokenPanelTitle: "Copy this token now",
      tokenPanelCopy: "This is the only time Relay will show the raw token.",
      tokenLabel: "Raw token",
      copyButton: "Copy token",
      copiedButton: "Copied",
      copyError: "Could not copy the token automatically.",
      listTitle: "Issued keys",
      emptyState: "No Relay API keys have been issued for this user yet.",
      scopeLabel: "Scope",
      projectLabel: "Project",
      activeStatus: "Active",
      revokedStatus: "Revoked",
      scopeGlobal: "Global",
      scopeProject: "Project",
      revokeButton: "Revoke",
      confirmRevokeButton: "Confirm revoke",
      cancelRevokeButton: "Cancel",
      revokeConfirmCopy: "This revokes the key immediately.",
      revokingButton: "Revoking...",
      fallbackIssueError: "Could not issue API key.",
      fallbackRevokeError: "Could not revoke API key.",
    },
    errorMap: {
      UNAUTHENTICATED: "Sign in again to manage Relay API keys.",
      INVALID_INPUT: "Enter a name before issuing a key.",
      API_KEY_NOT_FOUND_BY_ID: "That API key no longer exists.",
    },
  },
};
