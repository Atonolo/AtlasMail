'use strict';

function pressKey(window, keyCode, modifiers = []) {
  window.webContents.sendInputEvent({ type: 'keyDown', keyCode, modifiers });
  if (keyCode === 'Enter') window.webContents.sendInputEvent({ type: 'char', keyCode: '\r', modifiers });
  window.webContents.sendInputEvent({ type: 'keyUp', keyCode, modifiers });
}

async function tabTo(window, selector, modifiers = []) {
  pressKey(window, 'Tab', modifiers);
  let focus;
  for (let attempt = 0; attempt < 100; attempt += 1) {
    focus = await window.webContents.executeJavaScript(`(() => {
      const element = document.activeElement;
      const style = getComputedStyle(element);
      return {
        expected: element.matches(${JSON.stringify(selector)}),
        visible: element.matches(':focus-visible') && style.outlineStyle !== 'none' && parseFloat(style.outlineWidth) > 0,
        actual: element.id || element.className || element.tagName,
      };
    })()`);
    if (focus.expected && focus.visible) return;
    await new Promise((resolve) => setTimeout(resolve, 50));
  }
  throw new Error(`Keyboard focus did not reach ${selector} with a visible indicator (focused: ${focus.actual}, window: ${window.isFocused()}, contents: ${window.webContents.isFocused()})`);
}

function typeText(window, text) {
  for (const character of text) window.webContents.sendInputEvent({ type: 'char', keyCode: character });
}

async function startFromSkipLink(window) {
  window.show();
  window.focus();
  window.webContents.focus();
  await waitFor(window, 'document.hasFocus()', 'Timed out waiting for the keyboard window to focus');
  await window.webContents.executeJavaScript("document.querySelector('.skip-link').focus()");
  await tabTo(window, '.brand');
  await tabTo(window, '.nav-current');
  await tabTo(window, '#language-de');
}

async function exerciseLanguage(window) {
  await startFromSkipLink(window);
  await tabTo(window, '#language-en');
  pressKey(window, 'Enter');
  await waitFor(window, "document.documentElement.lang === 'en'", 'Keyboard language selection failed');
  await tabTo(window, '#language-de', ['shift']);
  pressKey(window, 'Enter');
  await waitFor(window, "document.documentElement.lang === 'de'", 'Keyboard language selection failed');
}

async function connect(window, token) {
  await startFromSkipLink(window);
  await tabTo(window, '#language-en');
  await tabTo(window, '#token');
  typeText(window, token);
  await tabTo(window, '#login-form button[type=submit]');
  pressKey(window, 'Enter');
}

async function addAccount(window, label, address) {
  await startFromSkipLink(window);
  await tabTo(window, '#language-en');
  await tabTo(window, '#account-name');
  typeText(window, label);
  await tabTo(window, '#account-address');
  typeText(window, address);
  await tabTo(window, '#account-protocol');
  await tabTo(window, '#save-account');
  pressKey(window, 'Enter');
}

async function waitFor(window, condition, message) {
  await window.webContents.executeJavaScript(`(async () => {
    for (let n = 0; n < 100; n++) {
      if (${condition}) return;
      await new Promise(resolve => setTimeout(resolve, 50));
    }
    throw new Error(${JSON.stringify(message)});
  })()`);
}

module.exports = { addAccount, connect, exerciseLanguage, waitFor };
