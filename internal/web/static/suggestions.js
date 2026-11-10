// A choice under a field, marked data-suggestion, writes itself into the field of the data-suggestions block it is in.
// The input event makes productfields.js take it as typed.
document.addEventListener("click", (event) => {
  const choice = event.target.closest("[data-suggestion]");
  if (!choice) return;
  const field = choice.closest("[data-suggestions]").querySelector("input");
  field.value = choice.textContent;
  field.dispatchEvent(new Event("input", { bubbles: true }));
});
