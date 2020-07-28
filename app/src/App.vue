<template>
    <div id="map">
        <l-map
                ref="map"
                :zoom.sync="zoom"
                :center="center"
                :options="mapOptions"
                @click="addLocation"
                @update:zoom="fetchGrid"
                @update:bounds="fetchGrid"
        >
            <l-tile-layer
                    :url="url"
                    :attribution="attribution"
            />
            <l-marker v-for="(latlng, index) in locations" :lat-lng="latlng" :key="index" ></l-marker>
            <l-circle-marker
                    :lat-lng="circle.center"
                    :radius="circle.radius"
                    :color="circle.color"
            />
            <l-geo-json :geojson="grid" :options="gridOptions"></l-geo-json>
        </l-map>
    </div>
</template>

<script>
    import {latLng} from 'leaflet';
    import {LGeoJson, LMap, LTileLayer, LMarker, LCircleMarker} from 'vue2-leaflet';
    import axios from 'axios'

    const host = 'http://localhost:8081'
    const urlLocations = `${host}/locations`
    const urlGrid = `${host}/grid`

    export default {
        name: "App",
        components: {
            LMap,
            LTileLayer,
            LGeoJson,
            LMarker,
            LCircleMarker
        },
        data() {
            return {
                center: latLng(-33.862451199999995, 151.207752),
                url: 'https://{s}.basemaps.cartocdn.com/light_all/{z}/{x}/{y}{r}.png',
                attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors &copy; <a href="https://carto.com/attributions">CARTO</a>',
                subdomains: 'abcd',
                zoom: 19,
                mapOptions: {
                    zoomSnap: 0.5,
                    zoomControl: false,
                },
                timer: null,
                locations: [],
                grid: null,
                gridOptions: {
                    color: "#000",
                    weight: 1,
                    opacity: 0.2,
                    fillOpacity: 0
                },
                circle: {}
            };
        },
        mounted() {
            this.$refs.map.mapObject.zoomControl = false
            this.fetchGrid()
            this.fetchLocations()
            this.timer = setInterval(this.fetchLocations, 1000)
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
                    this.grid = res.data.data || {}
                    console.log(res.data.data)
                })
            },
            fetchLocations() {
                axios.get(urlLocations).then((res) => {
                    const features = res.data.data.features || []
                    this.locations = features.map((feature) => {
                        const {
                            geometry: { coordinates }
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
                        geometry: { coordinates },
                        properties
                    } = res.data.data || {}

                    this.circle = {
                        center: latLng(coordinates[1], coordinates[0]),
                        radius: properties.radius,
                        color: properties.unique ? '#00ff00' : '#ff0000'
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
