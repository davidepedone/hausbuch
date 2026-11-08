// Formatting buttons for the page editor. Each function in `edits` takes the body, the selection's start and end, and
// returns the edit to make: replace start..end with insert, then select selStart..selEnd; or null for no edit.
const edits = {
  wrap(text, start, end, before, after) {
    const inner = text.slice(start, end);
    const at = start + before.length;
    return { start, end, insert: before + inner + after, selStart: at, selEnd: at + inner.length };
  },
  bold: (text, start, end) => edits.wrap(text, start, end, "**", "**"),
  pageLink: (text, start, end) => edits.wrap(text, start, end, "[[", "]]"),

  // withScheme gives an address typed without a scheme, such as example.com, https://.
  withScheme: (url) => (/^[a-z][a-z0-9+.-]*:/i.test(url) ? url : "https://" + url),

  // link links the selection to url, or inserts the address as its own text; url is null when the prompt was cancelled.
  link(text, start, end, url) {
    url = (url || "").trim();
    if (!url) return null;
    url = edits.withScheme(url);
    const insert = "[" + (text.slice(start, end) || url) + "](" + url + ")";
    const after = start + insert.length;
    return { start, end, insert, selStart: after, selEnd: after };
  },

  // linePrefix puts prefix at the start of every line the selection touches, or of the cursor's line.
  linePrefix(text, start, end, prefix) {
    const from = text.lastIndexOf("\n", start - 1) + 1;
    // A selection ending just after a line break doesn't touch the next line.
    let to = text.indexOf("\n", end > start ? end - 1 : end);
    if (to < 0) to = text.length;
    const insert = text.slice(from, to).split("\n").map((line) => prefix + line).join("\n");
    if (start === end) return { start: from, end: to, insert, selStart: start + prefix.length, selEnd: end + prefix.length };
    return { start: from, end: to, insert, selStart: from, selEnd: from + insert.length };
  },

  // pastedLink returns the pasted text when it is a single web address, "" otherwise.
  pastedLink(pasted) {
    pasted = pasted.trim();
    return /^https?:\/\/\S+$/i.test(pasted) ? pasted : "";
  },
};

if (typeof document !== "undefined") {
  const actions = {
    link: (text, start, end) => edits.link(text, start, end, prompt("Address of the link")),
    page: edits.pageLink,
    bold: edits.bold,
    heading: (text, start, end) => edits.linePrefix(text, start, end, "## "),
    list: (text, start, end) => edits.linePrefix(text, start, end, "- "),
    checklist: (text, start, end) => edits.linePrefix(text, start, end, "- [ ] "),
  };

  // apply goes through insertText so that the browser's undo can take the edit back.
  const apply = (area, edit) => {
    if (!edit) return;
    area.focus();
    area.setSelectionRange(edit.start, edit.end);
    if (!document.execCommand("insertText", false, edit.insert)) area.setRangeText(edit.insert);
    area.setSelectionRange(edit.selStart, edit.selEnd);
  };

  // Listening on the document keeps working after htmx swaps in a new body.
  document.addEventListener("click", (event) => {
    const button = event.target.closest("[data-format]");
    if (!button) return;
    const area = button.form.elements.body;
    apply(area, actions[button.dataset.format](area.value, area.selectionStart, area.selectionEnd));
  });

  document.addEventListener("paste", (event) => {
    const area = event.target;
    if (area.name !== "body" || area.selectionStart === area.selectionEnd) return;
    const url = edits.pastedLink(event.clipboardData.getData("text/plain"));
    if (!url) return;
    event.preventDefault();
    apply(area, edits.link(area.value, area.selectionStart, area.selectionEnd, url));
  });
}
