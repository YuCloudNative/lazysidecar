apiVersion: networking.istio.io/v1alpha3
kind: EnvoyFilter
metadata:
  name: {{ .Name }}
  namespace: {{ .Namespace }}
spec:
  configPatches:
    - applyTo: LISTENER
      match:
        context: SIDECAR_OUTBOUND
        listener:
          name: virtualOutbound
      patch:
        operation: MERGE
        value:
          listener_filters:
            - name: envoy.filters.listener.original_dst
              typed_config:
                "@type": type.googleapis.com/envoy.extensions.filters.listener.original_dst.v3.OriginalDst
            - name: envoy.filters.listener.tls_inspector
              typed_config:
                "@type": type.googleapis.com/envoy.extensions.filters.listener.tls_inspector.v3.TlsInspector
              {{ if gt (len .Services) 1 }}
              filter_disabled:
                or_match:
                  rules:
                  {{ range .Services }}
                  - destination_port_range:
                      start: {{ .Port }}
                      end: {{ inc .Port 1 }}
                  {{ end }}
              {{ else if eq (len .Services) 1 }}
              filter_disabled:
                destination_port_range:
                  start: {{ (index .Services 0).Port }}
                  end: {{ inc (index .Services 0).Port }}
              {{ end }}
            - name: envoy.filters.listener.http_inspector
              typed_config:
                "@type": type.googleapis.com/envoy.extensions.filters.listener.http_inspector.v3.HttpInspector
              {{ if gt (len .Services) 1 }}
              filter_disabled:
                or_match:
                  rules:
                  {{ range .Services }}
                  - destination_port_range:
                      start: {{ .Port }}
                      end: {{ inc .Port 1 }}
                  {{ end }}
              {{ else if eq (len .Services) 1 }}
              filter_disabled:
                destination_port_range:
                  start: {{ (index .Services 0).Port }}
                  end: {{ inc (index .Services 0).Port }}
              {{ end }}
    - applyTo: FILTER_CHAIN
      match:
        context: SIDECAR_OUTBOUND
        listener:
          filterChain:
            name: virtualOutbound-catchall-tcp
          name: virtualOutbound
      patch:
        operation: REMOVE
    - applyTo: FILTER_CHAIN
      match:
        context: SIDECAR_OUTBOUND
        listener:
          name: virtualOutbound
      patch:
        operation: ADD
        value:
          filter_chain_match:
            application_protocols:
              - http/1.1
              - h2
              - http/1.0
            transport_protocol: raw_buffer
          filters:
            - name: envoy.filters.network.http_connection_manager
              typed_config:
                "@type": type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
                codec_type: AUTO
                http_filters:
                  - name: istio.metadata_exchange
                    typed_config:
                      "@type": type.googleapis.com/udpa.type.v1.TypedStruct
                      type_url: type.googleapis.com/envoy.extensions.filters.http.wasm.v3.Wasm
                      value:
                        config:
                          configuration:
                            "@type": type.googleapis.com/google.protobuf.StringValue
                            value: |
                              {}
                          vm_config:
                            code:
                              local:
                                inline_string: envoy.wasm.metadata_exchange
                            runtime: envoy.wasm.runtime.null
                  - name: istio.alpn
                    typed_config:
                      "@type": type.googleapis.com/istio.envoy.config.filter.http.alpn.v2alpha1.FilterConfig
                      alpn_override:
                        - alpn_override:
                            - istio-http/1.0
                            - istio
                            - http/1.0
                        - alpn_override:
                            - istio-http/1.1
                            - istio
                            - http/1.1
                          upstream_protocol: HTTP11
                        - alpn_override:
                            - istio-h2
                            - istio
                            - h2
                          upstream_protocol: HTTP2
                  - name: envoy.filters.http.cors
                    typed_config:
                      "@type": type.googleapis.com/envoy.extensions.filters.http.cors.v3.Cors
                  - name: envoy.filters.http.fault
                    typed_config:
                      "@type": type.googleapis.com/envoy.extensions.filters.http.fault.v3.HTTPFault
                  - name: istio.stats
                    typed_config:
                      "@type": type.googleapis.com/udpa.type.v1.TypedStruct
                      type_url: type.googleapis.com/envoy.extensions.filters.http.wasm.v3.Wasm
                      value:
                        config:
                          configuration:
                            "@type": type.googleapis.com/google.protobuf.StringValue
                            value: |
                              {
                                "debug": "false",
                                "stat_prefix": "istio"
                              }
                          root_id: stats_outbound
                          vm_config:
                            code:
                              local:
                                inline_string: envoy.wasm.stats
                            runtime: envoy.wasm.runtime.null
                            vm_id: stats_outbound
                  - name: envoy.filters.http.lua
                    typed_config:
                      "@type": type.googleapis.com/envoy.extensions.filters.http.lua.v3.Lua
                      inline_code: |-
                        function envoy_on_request(request_handle)
                          request_handle:headers():add("src",  request_handle:headers():get("x-envoy-peer-metadata"))
                        end
                  - name: envoy.filters.http.router
                    typed_config:
                      "@type": type.googleapis.com/envoy.extensions.filters.http.router.v3.Router
                route_config:
                  name: csm-egressgateway
                  validate_clusters: false
                  virtual_hosts:
                    - domains:
                        - "*"
                      name: csm-egressgateway
                      routes:
                        - decorator:
                            operation: istio-egressgateway.istio-system.svc.cluster.local:80/*
                          match:
                            prefix: /
                          route:
                            cluster: outbound|{{ .LazysidecarGatewayPort }}||{{ .LazysidecarGateway }}.istio-system.svc.cluster.local
                stat_prefix: csm_ingress_http
                use_remote_address: false
          name: csm-egress
    - applyTo: FILTER_CHAIN
      match:
        context: SIDECAR_OUTBOUND
        listener:
          name: virtualOutbound
      patch:
        operation: ADD
        value:
          filters:
            - name: envoy.filters.network.tcp_proxy
              typed_config:
                "@type": type.googleapis.com/envoy.extensions.filters.network.tcp_proxy.v3.TcpProxy
                cluster: PassthroughCluster
                stat_prefix: CSMPassthroughCluster
          name: csm-passthrough
  workloadSelector:
    labels:
      {{ range $key, $value := .WorkloadSelector }}
      {{ $key }}: {{ $value }}
      {{ end }}