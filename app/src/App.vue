<template>
    <div id="map">
        <div class="floater" v-html="info"></div>
        <l-map
                ref="map"
                :zoom.sync="map.zoom"
                :center="map.center"
                :options="map.options"
                @click="addLocation"
                @update:zoom="fetchGrid"
                @update:bounds="fetchGrid"
        >
            <l-tile-layer
                    :url="map.url"
                    :attribution="map.attribution"
            />
            <l-marker v-for="(ll, index) in markers" :lat-lng="ll" :key="index"></l-marker>
            <l-circle
                    :lat-lng="distance.center"
                    :radius="distance.radius"
                    :color="distance.color"
            />
            <l-geo-json :geojson="grid.geoJSON" :options="grid.options"></l-geo-json>
            <l-geo-json :geojson="cells.geoJSON" :options="cells.options"></l-geo-json>
        </l-map>
        <notifications position="top right" group="info"/>
    </div>
</template>

<script>
    import {latLng} from 'leaflet';
    import {LCircle, LGeoJson, LMap, LMarker, LTileLayer} from 'vue2-leaflet';
    import axios from 'axios'

    const host = 'http://localhost:8081'
    const urlLocations = `${host}/locations`
    const urlGrid = `${host}/grid`
    const urlInfo = `${host}/info`
    const locUnique = '#00ff00'
    const locDuplicate = '#ff0000'

    export default {
        name: "App",
        components: {
            LMap,
            LTileLayer,
            LGeoJson,
            LMarker,
            LCircle
        },
        data() {
            return {
                map: {
                    center: latLng(-33.862451199999995, 151.207752),
                    url: 'https://{s}.basemaps.cartocdn.com/light_all/{z}/{x}/{y}{r}.png',
                    attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors &copy; <a href="https://carto.com/attributions">CARTO</a>',
                    subdomains: 'abcd',
                    zoom: 19,
                    options: {
                        zoomSnap: 0.5,
                        zoomControl: false,
                    },
                },
                timer: null,
                markers: [],
                distance: {},
                grid: {
                    geoJSON: null,
                    options: {
                        color: "#000",
                        weight: 1,
                        opacity: 0.2,
                        fillOpacity: 0
                    },
                },
                cells: {
                    geoJSON: null,
                    options: {
                        color: "#4e82de",
                        weight: 2,
                        opacity: 0.8,
                        fillOpacity: 0
                    },
                },
                toleranceTTL: '',
                toleranceDistance: '',
            };
        },
        mounted() {
            this.fetchInfo()
            this.fetchGrid()
        },
        methods: {
            fetchInfo() {
                axios.get(urlInfo).then((res) => {
                    const {distance, ttl} = res.data.data || {}

                    this.toleranceDistance = distance
                    this.toleranceTTL = ttl
                })
            },
            fetchGrid() {
                const bounds = this.$refs.map.mapObject.getBounds()
                axios.post(urlGrid, {
                    hi: bounds.getSouthWest(),
                    lo: bounds.getNorthEast()
                }).then((res) => {
                    this.grid.geoJSON = res.data.data || {}
                })
            },
            fetchLocations() {
                axios.get(urlLocations).then((res) => {
                    const features = res.data.data.features || []
                    this.markers = features.map((feature) => {
                        const {
                            geometry: {coordinates}
                        } = feature
                        return latLng(coordinates[1], coordinates[0])
                    })
                })
            },
            addLocation(ev) {
                axios.post(urlLocations, {
                    ...ev.latlng,
                    time: new Date()
                }).then((res) => {
                    const features = res.data.data.features || []

                    features.forEach((feature) => {
                        const {
                            geometry: {type, coordinates},
                            properties
                        } = feature

                        switch (type) {
                            case "Point":
                                this.distance = {
                                    center: latLng(coordinates[1], coordinates[0]),
                                    radius: properties.radius,
                                    color: properties.unique ? locUnique : locDuplicate
                                }
                                this.$notify({
                                    group: 'info',
                                    text: (properties.unique ? 'New location added' : 'Duplicate location detected'),
                                    type: (properties.unique ? 'info' : 'error'),
                                    duration: 3000,
                                    speed: 500
                                })
                                break
                            case "MultiPolygon":
                                this.cells.geoJSON = feature
                                break
                        }
                    })
                }).then(() => this.fetchLocations())
            }
        },
        computed: {
            info() {
                return `Distance tolerance: <strong>${this.toleranceDistance}</strong> meters<br />
                        Time tolerance: <strong>${this.toleranceTTL}</strong>`
            }
        }
    };
</script>

<style>
    body {
        padding: 0;
        margin: 0;
        font-family: system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, "Noto Sans", sans-serif;
        font-size: 1em;
    }

    html, body, #map {
        height: 100%;
        width: 100%;
    }

    .floater {
        display: inline-block;
        position: fixed;
        top: 0.5em;
        left: 0.5em;
        z-index: 10000;
        background: #fff;
        padding: 1em;
        border: 1px solid darkgrey;
    }
</style>
