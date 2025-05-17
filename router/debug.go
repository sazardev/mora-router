package router

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"strings"
)

// Debug creates a debugging middleware that adds request inspection
func WithDebug() Option {
	return func(r *MoraRouter) {
		r.middlewareRegistry["debug"] = debugMiddleware
		r.middlewares = append(r.middlewares, debugMiddleware)
		
		// Register inspector at /_mora/debug
		r.Get("/_mora/debug", r.debugHandler)
		r.Get("/_mora/routes", r.routesHandler)
		r.Get("/_mora/inspector", r.inspectorUI)
	}
}

// debugMiddleware loguea información detallada de las peticiones si se activa con la cabecera X-Mora-Debug
func debugMiddleware(next HandlerFunc) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, p Params) {
		if r.Header.Get("X-Mora-Debug") == "1" || r.URL.Query().Get("_debug") == "1" {
			// Add debug header to response
			w.Header().Set("X-Mora-Debug", "active")
			
			// Log detailed request info
			fmt.Printf("[MORA DEBUG] Request: %s %s\n", r.Method, r.URL.Path)
			fmt.Printf("[MORA DEBUG] Headers: %v\n", r.Header)
			fmt.Printf("[MORA DEBUG] Params: %v\n", p)
			fmt.Printf("[MORA DEBUG] Query: %v\n", r.URL.Query())
		}
		
		next(w, r, p)
	}
}

// routesHandler devuelve todas las rutas registradas en formato JSON
func (r *MoraRouter) routesHandler(w http.ResponseWriter, req *http.Request, p Params) {
	type RouteInfo struct {
		Method   string   `json:"method"`
		Pattern  string   `json:"pattern"`
		Segments []string `json:"segments"`
		Params   []string `json:"params"`
	}
	
	routes := make([]RouteInfo, 0, len(r.routes))
	for _, rt := range r.routes {
		params := []string{}
		segments := []string{}
		
		for _, seg := range rt.segments {
			if seg.name != "" {
				params = append(params, seg.name)
			}
			
			if seg.literal != "" {
				segments = append(segments, seg.literal)
			} else if seg.wildcard {
				segments = append(segments, "*"+seg.name)
			} else {
				var segDesc string
				if seg.regex != nil {
					segDesc = fmt.Sprintf(":%s(%s)", seg.name, seg.regex.String())
				} else {
					segDesc = ":" + seg.name
				}
				segments = append(segments, segDesc)
			}
		}
		
		routes = append(routes, RouteInfo{
			Method:   rt.method,
			Pattern:  rt.pattern,
			Segments: segments,
			Params:   params,
		})
	}
	
	// Sort routes by method and pattern for easier reading
	sort.Slice(routes, func(i, j int) bool {
		if routes[i].Method == routes[j].Method {
			return routes[i].Pattern < routes[j].Pattern
		}
		return routes[i].Method < routes[j].Method
	})
	
	JSON(w, http.StatusOK, routes)
}

// debugHandler muestra información detallada de la petición actual
func (r *MoraRouter) debugHandler(w http.ResponseWriter, req *http.Request, p Params) {
	debug := map[string]interface{}{
		"request": map[string]interface{}{
			"method":     req.Method,
			"path":       req.URL.Path,
			"query":      req.URL.Query(),
			"headers":    req.Header,
			"host":       req.Host,
			"remoteAddr": req.RemoteAddr,
			"params":     p,
		},
		"router": map[string]interface{}{
			"routeCount":        len(r.routes),
			"mountCount":        len(r.mounts),
			"middlewareCount":   len(r.middlewares),
			"registeredMacros":  len(MacroRegistry),
			"hasCustomNotFound": r.notFound != defaultNotFound,
		},
	}
	
	JSON(w, http.StatusOK, debug)
}

// inspectorUI devuelve una UI web para explorar las rutas y sus parámetros
func (r *MoraRouter) inspectorUI(w http.ResponseWriter, req *http.Request, p Params) {
	// HTML template for the inspector UI
	const tpl = `<!DOCTYPE html>
<html>
<head>
    <title>Mora Router Inspector</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
        }
        h1 {
            color: #0066cc;
            border-bottom: 1px solid #eee;
            padding-bottom: 10px;
        }
        table {
            width: 100%;
            border-collapse: collapse;
            margin: 20px 0;
        }
        th, td {
            padding: 12px 15px;
            border-bottom: 1px solid #ddd;
            text-align: left;
        }
        th {
            background-color: #f8f8f8;
            font-weight: bold;
        }
        tr:hover {
            background-color: #f5f5f5;
        }
        .method {
            font-weight: bold;
            width: 80px;
        }
        .get { color: #00aa00; }
        .post { color: #0000aa; }
        .put { color: #aa6600; }
        .delete { color: #aa0000; }
        .pattern { font-family: monospace; }
        .param { 
            display: inline-block;
            background-color: #f0f0f0;
            border-radius: 3px;
            padding: 2px 5px;
            margin: 2px;
            font-size: 0.85em;
        }
        #filter {
            width: 100%;
            padding: 8px;
            margin-bottom: 15px;
            border: 1px solid #ddd;
            border-radius: 4px;
            box-sizing: border-box;
        }
        .tabs {
            display: flex;
            margin-bottom: 20px;
            border-bottom: 1px solid #ddd;
        }
        .tab {
            padding: 10px 20px;
            cursor: pointer;
            border: 1px solid transparent;
            border-bottom: none;
        }
        .tab.active {
            border-color: #ddd;
            border-bottom-color: white;
            margin-bottom: -1px;
            background-color: white;
        }
        .tab-content {
            display: none;
        }
        .tab-content.active {
            display: block;
        }
        #debugInfo {
            background-color: #f8f8f8;
            padding: 15px;
            border-radius: 5px;
            font-family: monospace;
            white-space: pre-wrap;
        }
        .try-link {
            color: #0066cc;
            text-decoration: none;
            font-size: 0.9em;
            margin-left: 10px;
            cursor: pointer;
        }
        #requestForm {
            background-color: #f8f8f8;
            padding: 20px;
            border-radius: 5px;
            margin-bottom: 20px;
        }
        #requestForm label {
            display: block;
            margin-bottom: 5px;
        }
        #requestForm input, #requestForm select, #requestForm textarea {
            width: 100%;
            padding: 8px;
            margin-bottom: 10px;
            border: 1px solid #ddd;
            border-radius: 4px;
            box-sizing: border-box;
        }
        #requestForm button {
            background-color: #0066cc;
            color: white;
            padding: 10px 15px;
            border: none;
            border-radius: 4px;
            cursor: pointer;
        }
        #requestForm button:hover {
            background-color: #0052a3;
        }
    </style>
</head>
<body>
    <h1>Mora Router Inspector</h1>
    
    <div class="tabs">
        <div class="tab active" data-tab="routes">Routes</div>
        <div class="tab" data-tab="debug">Debug Info</div>
        <div class="tab" data-tab="request">Make Request</div>
    </div>
    
    <div id="routes" class="tab-content active">
        <input type="text" id="filter" placeholder="Filter routes...">
        <table>
            <thead>
                <tr>
                    <th>Method</th>
                    <th>Pattern</th>
                    <th>Parameters</th>
                    <th>Actions</th>
                </tr>
            </thead>
            <tbody id="routesTable">
                <tr><td colspan="4">Loading routes...</td></tr>
            </tbody>
        </table>
    </div>
    
    <div id="debug" class="tab-content">
        <div id="debugInfo">Loading debug information...</div>
    </div>
    
    <div id="request" class="tab-content">
        <div id="requestForm">
            <h3>Make a test request</h3>
            <label for="methodInput">Method:</label>
            <select id="methodInput">
                <option value="GET">GET</option>
                <option value="POST">POST</option>
                <option value="PUT">PUT</option>
                <option value="DELETE">DELETE</option>
                <option value="PATCH">PATCH</option>
                <option value="OPTIONS">OPTIONS</option>
            </select>
            
            <label for="pathInput">Path:</label>
            <input type="text" id="pathInput" placeholder="/api/resource/:id">
            
            <label for="headersInput">Headers (JSON):</label>
            <textarea id="headersInput" rows="3" placeholder='{"Content-Type": "application/json"}'></textarea>
            
            <label for="bodyInput">Body (for POST, PUT, PATCH):</label>
            <textarea id="bodyInput" rows="5" placeholder='{"name": "value"}'></textarea>
            
            <button id="sendRequest">Send Request</button>
        </div>
        
        <h3>Response</h3>
        <div id="responseInfo">
            <p>Send a request to see the response here.</p>
        </div>
    </div>

    <script>
        // Fetch and display routes
        fetch('/_mora/routes')
            .then(response => response.json())
            .then(routes => {
                const table = document.getElementById('routesTable');
                table.innerHTML = '';
                
                routes.forEach(route => {
                    const tr = document.createElement('tr');
                    
                    // Method cell
                    const methodCell = document.createElement('td');
                    methodCell.textContent = route.method;
                    methodCell.className = 'method ' + route.method.toLowerCase();
                    tr.appendChild(methodCell);
                    
                    // Pattern cell
                    const patternCell = document.createElement('td');
                    patternCell.textContent = route.pattern;
                    patternCell.className = 'pattern';
                    tr.appendChild(patternCell);
                    
                    // Parameters cell
                    const paramsCell = document.createElement('td');
                    route.params.forEach(param => {
                        const span = document.createElement('span');
                        span.textContent = param;
                        span.className = 'param';
                        paramsCell.appendChild(span);
                    });
                    tr.appendChild(paramsCell);
                    
                    // Actions cell
                    const actionsCell = document.createElement('td');
                    if (route.method === 'GET') {
                        const tryLink = document.createElement('a');
                        tryLink.textContent = 'Try';
                        tryLink.className = 'try-link';
                        tryLink.onclick = () => {
                            document.querySelector('[data-tab="request"]').click();
                            document.getElementById('methodInput').value = 'GET';
                            document.getElementById('pathInput').value = route.pattern;
                        };
                        actionsCell.appendChild(tryLink);
                    }
                    tr.appendChild(actionsCell);
                    
                    table.appendChild(tr);
                });
            })
            .catch(error => {
                console.error('Error fetching routes:', error);
                document.getElementById('routesTable').innerHTML = '<tr><td colspan="4">Error loading routes</td></tr>';
            });
            
        // Fetch and display debug info
        fetch('/_mora/debug')
            .then(response => response.json())
            .then(debug => {
                document.getElementById('debugInfo').textContent = JSON.stringify(debug, null, 2);
            })
            .catch(error => {
                console.error('Error fetching debug info:', error);
                document.getElementById('debugInfo').textContent = 'Error loading debug information';
            });
            
        // Filter routes
        document.getElementById('filter').addEventListener('input', function(e) {
            const filter = e.target.value.toLowerCase();
            const rows = document.getElementById('routesTable').getElementsByTagName('tr');
            
            for (let i = 0; i < rows.length; i++) {
                const method = rows[i].getElementsByClassName('method')[0];
                const pattern = rows[i].getElementsByClassName('pattern')[0];
                
                if (method && pattern) {
                    const text = method.textContent.toLowerCase() + ' ' + pattern.textContent.toLowerCase();
                    if (text.includes(filter)) {
                        rows[i].style.display = '';
                    } else {
                        rows[i].style.display = 'none';
                    }
                }
            }
        });
        
        // Tab switching
        document.querySelectorAll('.tab').forEach(tab => {
            tab.addEventListener('click', () => {
                document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
                document.querySelectorAll('.tab-content').forEach(c => c.classList.remove('active'));
                
                tab.classList.add('active');
                document.getElementById(tab.getAttribute('data-tab')).classList.add('active');
            });
        });
        
        // Send test request
        document.getElementById('sendRequest').addEventListener('click', () => {
            const method = document.getElementById('methodInput').value;
            const path = document.getElementById('pathInput').value;
            let headers = {};
            
            try {
                const headersText = document.getElementById('headersInput').value.trim();
                if (headersText) {
                    headers = JSON.parse(headersText);
                }
            } catch (e) {
                alert('Invalid headers JSON: ' + e.message);
                return;
            }
            
            let body = document.getElementById('bodyInput').value;
            if (body && (method === 'POST' || method === 'PUT' || method === 'PATCH')) {
                headers['Content-Type'] = headers['Content-Type'] || 'application/json';
            } else {
                body = null;
            }
            
            const responseInfo = document.getElementById('responseInfo');
            responseInfo.innerHTML = '<p>Sending request...</p>';
            
            fetch(path, {
                method: method,
                headers: headers,
                body: body
            })
            .then(async response => {
                const responseBody = await response.text();
                let formattedBody = responseBody;
                
                try {
                    // If it's JSON, format it
                    const json = JSON.parse(responseBody);
                    formattedBody = JSON.stringify(json, null, 2);
                } catch (e) {
                    // Not JSON, use as is
                }
                
                const headersList = Array.from(response.headers.entries())
                    .map(([key, value]) => `<strong>${key}:</strong> ${value}`)
                    .join('<br>');
                
                responseInfo.innerHTML = `
                    <h4>Status: ${response.status} ${response.statusText}</h4>
                    <div>
                        <h4>Headers:</h4>
                        <div>${headersList}</div>
                    </div>
                    <div>
                        <h4>Body:</h4>
                        <pre style="background: #f8f8f8; padding: 10px; overflow: auto; max-height: 300px;">${formattedBody}</pre>
                    </div>
                `;
            })
            .catch(error => {
                responseInfo.innerHTML = `<p>Error: ${error.message}</p>`;
            });
        });
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl, err := template.New("inspector").Parse(tpl)
	if err != nil {
		http.Error(w, "Error rendering inspector UI", http.StatusInternalServerError)
		return
	}
	
	err = tmpl.Execute(w, nil)
	if err != nil {
		http.Error(w, "Error rendering inspector UI", http.StatusInternalServerError)
	}
}

// DebugPrint imprime información de depuración si el modo Debug está activado
func DebugPrint(format string, args ...interface{}) {
	if os.Getenv("MORA_DEBUG") == "1" {
		fmt.Printf("[MORA DEBUG] "+format+"\n", args...)
	}
}
