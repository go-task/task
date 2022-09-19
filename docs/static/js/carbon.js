(function () {
  function attachAd() {
    var el = document.createElement('script');
    el.setAttribute('type', 'text/javascript');
    el.setAttribute('id', '_carbonads_js');
    el.setAttribute(
      'src',
      '//cdn.carbonads.com/carbon.js?serve=CESI65QJ&placement=taskfiledev'
    );
    el.setAttribute('async', 'async');

    var wrapper = document.getElementById('sidebar-ads');
    wrapper.innerHTML = '';
    wrapper.appendChild(el);
  }

  setTimeout(function () {
    attachAd();

    var currentPath = window.location.pathname;

    setInterval(function () {
      if (currentPath !== window.location.pathname) {
        currentPath = window.location.pathname;
        attachAd();
      }
    }, 1000);
  }, 1000);
})();
