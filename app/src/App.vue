<template>
    <div id="map">
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
        </l-map>
    </div>
</template>

<script>
    import {latLng} from 'leaflet';
    import {LCircle, LGeoJson, LMap, LMarker, LTileLayer} from 'vue2-leaflet';
    import axios from 'axios'

    const host = 'http://localhost:8081'
    const urlLocations = `${host}/locations`
    const urlGrid = `${host}/grid`
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
            };
        },
        mounted() {
            this.fetchGrid()
            this.fetchLocations()
            this.timer = setInterval(this.fetchLocations, 500)
        },
        beforeDestroy() {
            clearInterval(this.timer)
        },
        methods: {
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
                    ...ev.latlng
                }).then((res) => {
                    const {
                        geometry: {coordinates},
                        properties
                    } = res.data.data || {}
                    this.distance = {
                        center: latLng(coordinates[1], coordinates[0]),
                        radius: properties.radius,
                        color: properties.unique ? locUnique : locDuplicate
                    }
                })
            }
        }
    };
</script>

<style>
    body {
        padding: 0;
        margin: 0;
    }

    html, body, #map {
        height: 100%;
        width: 100%;
    }
</style>
