// X-SL UI initialization
document.addEventListener("DOMContentLoaded", function () {
  // Initialize scroll animations
  if (window.AOS) {
    AOS.init({
      duration: 450,
      once: true,
      offset: 40,
      easing: "ease-out-cubic"
    });
  }

  // Theme sync: detect existing theme and set data-theme attribute
  function syncTheme() {
    var bodyClass = document.body.className || '';
    var htmlClass = document.documentElement.className || '';
    var theme = 'dark'; // default
    
    if (bodyClass.includes('light') || htmlClass.includes('light')) {
      theme = 'light';
    } else if (bodyClass.includes('dark') || htmlClass.includes('dark')) {
      theme = 'dark';
    }
    
    document.documentElement.setAttribute('data-theme', theme);
    localStorage.setItem('xsl-theme', theme);
  }
  
  syncTheme();
  
  // Observe body class changes to keep theme synced
  var observer = new MutationObserver(function(mutations) {
    syncTheme();
  });
  observer.observe(document.body, { attributes: true, attributeFilter: ['class'] });
  observer.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] });

  // Expose icon helper globally for inline HTML
  window.xslIcon = function (name, size) {
    return '<iconify-icon icon="' + name + '" width="' + (size || 18) + '"></iconify-icon>';
  };

  console.log("%c X-SL Premium UI loaded ", "background: #6366f1; color: white; padding: 4px 8px; border-radius: 4px;");
});