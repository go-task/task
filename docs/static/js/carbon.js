(function () {
  function attachAd() {
    const el = document.createElement('script');
    el.setAttribute('type', 'text/javascript');
    el.setAttribute('id', '_carbonads_js');
    el.setAttribute(
      'src',
      '//cdn.carbonads.com/carbon.js?serve=CESI65QJ&placement=taskfiledev'
    );
    el.setAttribute('async', 'async');

    const wrapper = document.getElementById('sidebar-ads');
    wrapper.innerHTML = '';
    wrapper.appendChild(el);
  }

  setTimeout(function () {
    attachAd();

    window.addEventListener('popstate', function () {
      attachAd();
    });
  }, 1000);
})();
