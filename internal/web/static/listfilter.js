// The chips at the top of a list. The markup marks the filter's box with data-filter, its chips with data-filter-chip
// (the key of the entries they stand for) and its "All" chip with data-filter-all; each entry with data-entry and
// data-chip; each group of entries with data-group; and the line saying that nothing matches with data-filter-none.
{
  // The chips may come with the list, so the box remembers the chosen chip's key, falling back to "All" when a new
  // list no longer has that chip.
  const apply = (box) => {
    const chips = [...box.querySelectorAll("[data-filter-chip]")];
    const chip = chips.find((c) => c.dataset.filterChip === box.dataset.chosen);
    for (const c of box.querySelectorAll("[data-filter-chip], [data-filter-all]")) {
      c.setAttribute("aria-pressed", chip ? c === chip : c.hasAttribute("data-filter-all"));
    }
    const entries = box.querySelectorAll("[data-entry]");
    let shown = 0;
    for (const entry of entries) {
      entry.hidden = !!chip && entry.dataset.chip !== chip.dataset.filterChip;
      if (!entry.hidden) shown++;
    }
    for (const group of box.querySelectorAll("[data-group]")) {
      group.hidden = !group.querySelector("[data-entry]:not([hidden])");
    }
    // An empty list says so itself.
    box.querySelector("[data-filter-none]").hidden = shown > 0 || entries.length === 0;
  };
  const applyAll = () => document.querySelectorAll("[data-filter]").forEach(apply);

  document.addEventListener("click", (event) => {
    const chosen = event.target.closest("[data-filter-chip], [data-filter-all]");
    if (!chosen) return;
    const box = chosen.closest("[data-filter]");
    if (chosen.hasAttribute("data-filter-all")) delete box.dataset.chosen;
    else box.dataset.chosen = chosen.dataset.filterChip;
    apply(box);
  });
  // A row's buttons swap in a new list, which comes unfiltered.
  document.addEventListener("htmx:afterSwap", applyAll);
}
