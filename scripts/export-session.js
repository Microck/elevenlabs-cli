// Paste into DevTools on elevenlabs.io after signing in. Treat the downloaded
// file as a password. Import with: elevenlabs auth login --session FILE.
(() => {
  if (location.origin !== 'https://elevenlabs.io') throw new Error('Open elevenlabs.io first');
  const entries = Object.keys(localStorage).filter(key => key.startsWith('firebase:authUser:'));
  if (entries.length !== 1) throw new Error('Expected one signed-in ElevenLabs account');
  const user = JSON.parse(localStorage.getItem(entries[0]));
  const tokens = user.stsTokenManager;
  if (!tokens?.refreshToken || !tokens?.accessToken) throw new Error('Sign in again before exporting');
  const session = {
    user_agent: navigator.userAgent,
    api_key: user.apiKey,
    refresh_token: tokens.refreshToken,
    access_token: tokens.accessToken,
    expires_at: new Date(tokens.expirationTime).toISOString()
  };
  const url = URL.createObjectURL(new Blob([JSON.stringify(session)], {type: 'application/json'}));
  const link = document.createElement('a');
  link.href = url;
  link.download = 'elevenlabs-session.json';
  link.click();
  setTimeout(() => URL.revokeObjectURL(url), 1000);
})();
