window.customilyShopify = window.customilyShopify || {};
window.customilyShopify.hooks = {
  onAppLoaded: function () {
    if (!window?.customilyAppLoaded) {
      console.log("================= Mark app loaded =================");
      window.customilyAppLoaded = true;
    }
  },
};
