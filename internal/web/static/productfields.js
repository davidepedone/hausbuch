// The add form's name field, marked with data-product-name, asks for the values a product last had once a name is
// chosen. They fill in the form's fields that the user hasn't typed in, those in #product-fields and those marked
// data-product-field, so that an answer arriving late, on a slow connection, doesn't overwrite what was typed meanwhile.
{
  const typed = new WeakSet();
  // A form marked data-kept="location" holds the location of the item just added, which stays for the next one.
  for (const form of document.querySelectorAll("form[data-kept]")) typed.add(form.elements[form.dataset.kept]);
  document.addEventListener("input", (event) => {
    if (event.target.closest("#product-fields, [data-product-field]")) typed.add(event.target);
  });
  document.addEventListener("htmx:afterRequest", (event) => {
    const name = event.detail.elt;
    if (!name.matches("[data-product-name]") || !event.detail.successful) return;
    // A name never used answers with nothing, which leaves the fields as they are.
    const answer = new DOMParser().parseFromString(event.detail.xhr.responseText, "text/html");
    for (const value of answer.querySelectorAll("[name]")) {
      const field = name.form.elements[value.name];
      if (field && !typed.has(field)) field.value = value.value;
    }
  });
}
