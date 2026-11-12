// An expiry field, marked data-expiry, shows either its month and year or its exact day; a data-expiry-switch button
// shows the other part. A hidden part's fields are disabled, so that the form posts only the shown one.
document.addEventListener("click", (event) => {
  const button = event.target.closest("[data-expiry-switch]");
  if (!button) return;
  for (const part of button.closest("[data-expiry]").querySelectorAll("[data-expiry-part]")) {
    part.hidden = !part.hidden;
    for (const field of part.querySelectorAll("select, input")) field.disabled = part.hidden;
    if (!part.hidden) part.querySelector("select, input").focus();
  }
});
