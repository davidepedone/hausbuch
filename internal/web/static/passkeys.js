// Passkey sign-in and creation. A [data-passkey-sign-in] block signs in with a passkey, from its button and from the
// browser's suggestions in the username field; a [data-passkey-create] block creates one for the signed-in user. Both
// go to their data-next path when done, and show what went wrong in their [data-passkey-error].
{
  const decode = (s) => Uint8Array.from(atob(s.replace(/-/g, "+").replace(/_/g, "/")), (c) => c.charCodeAt(0)).buffer;
  const encode = (b) =>
    btoa(String.fromCharCode(...new Uint8Array(b))).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");

  // The server's options name binary fields in base64url; the browser wants them as buffers.
  const creationOptions = (o) => ({
    ...o,
    challenge: decode(o.challenge),
    user: { ...o.user, id: decode(o.user.id) },
    excludeCredentials: (o.excludeCredentials || []).map((c) => ({ ...c, id: decode(c.id) })),
  });
  const requestOptions = (o) => ({
    ...o,
    challenge: decode(o.challenge),
    allowCredentials: (o.allowCredentials || []).map((c) => ({ ...c, id: decode(c.id) })),
  });

  const credentialJSON = (c) => {
    const r = c.response;
    const response = { clientDataJSON: encode(r.clientDataJSON) };
    if (r.attestationObject) {
      response.attestationObject = encode(r.attestationObject);
      response.transports = r.getTransports ? r.getTransports() : [];
    } else {
      response.authenticatorData = encode(r.authenticatorData);
      response.signature = encode(r.signature);
      if (r.userHandle) response.userHandle = encode(r.userHandle);
    }
    return { id: c.id, rawId: encode(c.rawId), type: c.type, response,
      clientExtensionResults: c.getClientExtensionResults() };
  };

  const post = async (url, body) => {
    const resp = await fetch(url, { method: "POST", headers: { "Content-Type": "application/json" },
      body: body && JSON.stringify(body) });
    const answer = await resp.json();
    if (!resp.ok) throw new Error(answer.error);
    return answer;
  };

  const show = (block, message) => {
    const p = block.querySelector("[data-passkey-error]");
    p.textContent = message;
    p.hidden = false;
  };

  const next = (block) => "?next=" + encodeURIComponent(block.dataset.next || "/");

  // signIn runs one passkey sign-in; conditional ones wait for the user to pick a passkey among the username field's
  // suggestions, and are cancelled when the button starts another. A conditional one that can't start, over the
  // request limit for one, says nothing: the user hasn't asked for anything yet.
  let pending;
  const signIn = async (block, conditional) => {
    pending?.abort();
    pending = new AbortController();
    try {
      const options = await post("/login/passkey/begin").catch((e) => {
        if (conditional) e.name = "AbortError";
        throw e;
      });
      const credential = await navigator.credentials.get({ publicKey: requestOptions(options.publicKey),
        signal: pending.signal, mediation: conditional ? "conditional" : "optional" });
      const answer = await post("/login/passkey/finish" + next(block), credentialJSON(credential));
      location.assign(answer.next);
    } catch (e) {
      if (e.name === "AbortError" && conditional) return;
      show(block, e.name === "Error" && e.message ? e.message
        : "The passkey sign-in didn't complete. Try again, or sign in with your password.");
    }
  };

  const create = async (block) => {
    try {
      const options = await post("/account/passkeys/begin");
      const credential = await navigator.credentials.create({ publicKey: creationOptions(options.publicKey) });
      const answer = await post("/account/passkeys/finish" + next(block), credentialJSON(credential));
      location.assign(answer.next);
    } catch (e) {
      show(block, e.name === "Error" && e.message ? e.message : "The passkey wasn't created. Try again.");
    }
  };

  for (const block of document.querySelectorAll("[data-passkey-sign-in]")) {
    block.querySelector("[data-passkey-button]").addEventListener("click", () => signIn(block, false));
    window.PublicKeyCredential?.isConditionalMediationAvailable?.().then((ok) => ok && signIn(block, true));
  }
  for (const block of document.querySelectorAll("[data-passkey-create]")) {
    block.querySelector("[data-passkey-button]").addEventListener("click", () => create(block));
  }
}
