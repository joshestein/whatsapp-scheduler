// Searchable combobox over the embedded contact list. Filtering is client-side.
// The hidden recipient_jid carries the chosen JID, or the raw text when nothing
// was chosen; the server normalises raw text.
(() => {
  const box = document.getElementById("recipient");
  const hidden = box.form.elements.recipient_jid;
  const list = document.getElementById(box.getAttribute("aria-controls"));
  const options = Array.from(list.querySelectorAll('[role="option"]'));
  let active = null;

  const visible = () => options.filter((o) => !o.hidden);

  function highlight(opt) {
    active?.removeAttribute("aria-selected");
    active = opt;
    if (opt) {
      opt.setAttribute("aria-selected", "true");
      opt.scrollIntoView({ block: "nearest" });
      box.setAttribute("aria-activedescendant", opt.id);
    } else {
      box.removeAttribute("aria-activedescendant");
    }
  }

  function open() {
    filter();
    list.hidden = false;
    box.setAttribute("aria-expanded", "true");
  }

  function close() {
    list.hidden = true;
    box.setAttribute("aria-expanded", "false");
    highlight(null);
  }

  // data-search is name + phone (people) or name (groups); never the JID.
  function filter() {
    const q = box.value.trim().toLowerCase();
    for (const o of options) o.hidden = !o.dataset.search.includes(q);
    highlight(visible()[0] ?? null);
  }

  function select(opt) {
    hidden.value = opt.dataset.jid;
    box.value = opt.dataset.name;
    box.parentElement.classList.add("selected");
    close();
  }

  box.addEventListener("focus", open);
  box.addEventListener("click", () => list.hidden && open());
  box.addEventListener("blur", close); // option mousedown is prevented, so this only fires on real focus loss
  box.addEventListener("input", () => {
    hidden.value = box.value;
    box.parentElement.classList.remove("selected");
    open();
  });

  box.addEventListener("keydown", (e) => {
    const isOpen = !list.hidden;
    switch (e.key) {
      case "ArrowDown":
      case "ArrowUp": {
        e.preventDefault();
        if (!isOpen) open();
        const vis = visible();
        if (!vis.length) return;
        const i = vis.indexOf(active) + (e.key === "ArrowDown" ? 1 : -1);
        highlight(vis[Math.max(0, Math.min(vis.length - 1, i))]);
        break;
      }
      case "Enter":
        if (!isOpen) return; // closed: normal submit
        e.preventDefault();
        if (active) select(active);
        else close();
        break;
      case "Escape":
        if (isOpen) close();
        break;
    }
  });

  list.addEventListener("mousedown", (e) => e.preventDefault()); // keep focus in the box
  list.addEventListener("click", (e) => {
    const opt = e.target.closest('[role="option"]');
    if (opt) select(opt);
  });

  // this.reset() clears values only. Filter and highlight are ours to clear.
  box.form.addEventListener("reset", () => {
    box.parentElement.classList.remove("selected");
    close();
  });
})();
