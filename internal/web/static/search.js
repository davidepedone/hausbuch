// Closes the panel of results under the header's search field on Escape and on a click outside the search form.
{
  const close = () => {
    const panel = document.getElementById("search-panel");
    if (panel) panel.replaceChildren();
  };
  document.addEventListener("keydown", (event) => {
    if (event.key === "Escape") close();
  });
  document.addEventListener("click", (event) => {
    if (!event.target.closest('form[role="search"]')) close();
  });
}
