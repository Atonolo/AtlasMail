'use strict';

if (window.opener) {
  window.opener.postMessage({
    type: 'atlas-mail-oauth-complete',
    success: document.body.dataset.oauthSuccess === 'true',
  }, window.location.origin);
}
