const PATH_STROKE_COLOR = "#FF0000";
const PATH_STROKE_WIDTH = 2;
const PATH_STROKE_OPACITY = 1.0;
const PATH_IS_GEODESIC = true;

class ActivityMap {
  constructor(mapelem, summaryelem) {
    this.map = new google.maps.Map(mapelem, {
      zoom: 11,
      center: {lat: 0.0, lng: 0.0},
      mapTypeId: "terrain",
    });
    this.summary = summaryelem;
    this.path = null;
  }

  clearPath() {
    if (this.path === null) {
      return
    }
    this.path.setMap(null);
  }

  clearInfo() {
    this.summary.innerHTML = "";
  }

  renderActivity(activity) {
    this.clearPath();
    let coords = activity.Coords.map(gps => ({
      lat: gps.Lat,
      lng: gps.Lng,
    }));
    this.path = new google.maps.Polyline({
      path: coords,
      geodesic: PATH_IS_GEODESIC,
      strokeColor: PATH_STROKE_COLOR,
      strokeOpacity: PATH_STROKE_OPACITY,
      strokeWeight: PATH_STROKE_WIDTH,
    });
    this.path.setMap(this.map);

    let bounds  = new google.maps.LatLngBounds();
    coords.forEach(c => bounds.extend(c));
    this.map.fitBounds(bounds);

    this.clearInfo();
    let that = this;
    const addInfo = function(info) {
      let elem = document.createElement("p");
      elem.textContent = info;
      that.summary.appendChild(elem);
    };

    const date = new Date(activity.Summary.Timestamp * 1000);
    addInfo(`${date}`);
    addInfo(`Sport: ${activity.Summary.Sport}`);
    addInfo(`Duration: ${activity.Summary.Duration}`);
    addInfo(`Distance: ${activity.Summary.Distance} km`);
  }
}

function initMap() {
  let map = new ActivityMap(
    document.getElementById("map"),
    document.getElementById("summary"),
  );

  const sel = document.getElementById("act");
  sel.addEventListener("change", (evt) => {
    const val = evt.target.value;
    if (val === "") {
      return;
    }
    fetch(`/activity?fit=${val}`)
      .then(response => response.json())
      .then(activity => map.renderActivity(activity));
  });
}
