document.querySelectorAll("[data-confirm]").forEach((form) => {
  form.addEventListener("submit", (event) => {
    if (!window.confirm(form.dataset.confirm)) event.preventDefault();
  });
});

const slugSource = document.querySelector("[data-slug-source]");
const slugTarget = document.querySelector("[data-slug-target]");
if (slugSource && slugTarget && !slugTarget.value) {
  slugSource.addEventListener("input", () => {
    slugTarget.value = slugSource.value
      .toLowerCase()
      .normalize("NFKD")
      .replace(/[^a-z0-9]+/g, "-")
      .replace(/(^-|-$)/g, "");
  });
}

document.querySelectorAll("[data-auto-submit]").forEach((field) => {
  field.addEventListener("change", () => field.form.submit());
});
