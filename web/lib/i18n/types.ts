export type Locale = "en" | "ko";

export type Dictionary = {
  common: {
    continueWithGitHub: string;
    unknownError: string;
    links: {
      backToOnboarding: string;
      projectExplorer: string;
      providerSettings: string;
      apiKeys: string;
      styleMemory: string;
    };
    language: {
      label: string;
      english: string;
      korean: string;
    };
  };
  root: {
    eyebrow: string;
    title: string;
    subtitle: string;
    panelTitle: string;
    panelCopy: string;
    signInButton: string;
  };
  onboarding: {
    page: {
      eyebrow: string;
      title: string;
      subtitle: string;
      signInTitle: string;
      signInCopy: string;
    };
    client: {
      signedInEyebrowPrefix: string;
      fallbackUser: string;
      title: string;
      copy: string;
      startButton: string;
      startingButton: string;
      styleMemoryLink: string;
      providerSettingsLink: string;
      apiKeysLink: string;
      fallbackError: string;
    };
  };
  providers: {
    page: {
      eyebrow: string;
      signInTitle: string;
      signInCopy: string;
      loadErrorTitle: string;
      loadErrorCopy: string;
    };
    client: {
      eyebrow: string;
      title: string;
      copy: string;
      settingsOnlyPill: string;
      connected: string;
      disconnected: string;
      noStoredKey: string;
      maskedKeySeparator: string;
      savingStatus: string;
      disconnectingStatus: string;
      connectedHelp: string;
      disconnectedHelp: string;
      apiKeyLabel: string;
      apiKeyPlaceholder: string;
      fieldHelp: string;
      connectButton: string;
      replaceButton: string;
      savingButton: string;
      disconnectButton: string;
      disconnectingButton: string;
      fallbackConnectError: string;
      fallbackDisconnectError: string;
    };
    errorMap: Record<string, string>;
  };
  apiKeys: {
    page: {
      eyebrow: string;
      signInTitle: string;
      signInCopy: string;
      loadErrorTitle: string;
      loadErrorCopy: string;
    };
    client: {
      eyebrow: string;
      title: string;
      copy: string;
      settingsOnlyPill: string;
      nameLabel: string;
      namePlaceholder: string;
      fieldHelp: string;
      issueButton: string;
      issuingButton: string;
      issuedSuccess: string;
      revokedSuccess: string;
      tokenPanelTitle: string;
      tokenPanelCopy: string;
      tokenLabel: string;
      copyButton: string;
      copiedButton: string;
      copyError: string;
      listTitle: string;
      emptyState: string;
      scopeLabel: string;
      projectLabel: string;
      activeStatus: string;
      revokedStatus: string;
      scopeGlobal: string;
      scopeProject: string;
      revokeButton: string;
      confirmRevokeButton: string;
      cancelRevokeButton: string;
      revokeConfirmCopy: string;
      revokingButton: string;
      fallbackIssueError: string;
      fallbackRevokeError: string;
    };
    errorMap: Record<string, string>;
  };
};
