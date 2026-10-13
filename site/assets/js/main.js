(function () {
  "use strict";

  function copyText(text) {
    if (navigator.clipboard && navigator.clipboard.writeText) {
      return navigator.clipboard.writeText(text);
    }
    return new Promise(function (resolve, reject) {
      var ta = document.createElement("textarea");
      ta.value = text;
      ta.setAttribute("readonly", "");
      ta.style.position = "absolute";
      ta.style.left = "-9999px";
      document.body.appendChild(ta);
      ta.select();
      try {
        document.execCommand("copy");
        resolve();
      } catch (err) {
        reject(err);
      } finally {
        document.body.removeChild(ta);
      }
    });
  }

  document.querySelectorAll("[data-copy]").forEach(function (btn) {
    btn.addEventListener("click", function () {
      var targetId = btn.getAttribute("data-copy");
      var el = targetId ? document.getElementById(targetId) : null;
      if (!el) {
        return;
      }
      var text = el.textContent.replace(/\n$/, "");
      var label = btn.textContent;
      copyText(text)
        .then(function () {
          btn.setAttribute("data-copied", "true");
          btn.textContent = "Copied";
          window.setTimeout(function () {
            btn.removeAttribute("data-copied");
            btn.textContent = label;
          }, 1600);
        })
        .catch(function () {
          btn.textContent = "Copy failed";
          window.setTimeout(function () {
            btn.textContent = label;
          }, 1600);
        });
    });
  });
})();
