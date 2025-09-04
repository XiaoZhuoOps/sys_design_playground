import React, { useState, useEffect } from 'react';
import axios from 'axios';
import ScenarioViewer from './components/ScenarioViewer';

// Configure axios to proxy requests to the backend container
const apiClient = axios.create({
    baseURL: '/api',
});

function App() {
    const [scenarios, setScenarios] = useState([]);
    const [selectedScenarioId, setSelectedScenarioId] = useState(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(null);

    // Fetch all available scenarios on component mount
    useEffect(() => {
        apiClient.get('/scenarios')
            .then(response => {
                setScenarios(response.data);
                setLoading(false);
            })
            .catch(err => {
                console.error("Failed to fetch scenarios:", err);
                setError("Could not connect to the backend. Please ensure it's running.");
                setLoading(false);
            });
    }, []);

    const handleMenuClick = ({ key }) => {
        setSelectedScenarioId(key);
    };

    return (
        <div style={{ display: 'flex', minHeight: '100vh' }}>
            <div style={{ width: '300px', borderRight: '1px solid #ddd', padding: '20px' }}>
                <h3>SYS DESIGN PLAYGROUND</h3>
                {loading ? (
                    <div>Loading...</div>
                ) : (
                    <ul style={{ listStyle: 'none', padding: 0 }}>
                        {scenarios.map(s => (
                            <li key={s.id} style={{ marginBottom: '10px' }}>
                                <button
                                    onClick={() => setSelectedScenarioId(s.id)}
                                    style={{
                                        width: '100%',
                                        padding: '10px',
                                        border: '1px solid #ddd',
                                        backgroundColor: selectedScenarioId === s.id ? '#f0f0f0' : 'white',
                                        cursor: 'pointer'
                                    }}
                                >
                                    {s.title}
                                </button>
                            </li>
                        ))}
                    </ul>
                )}
            </div>
            <div style={{ flex: 1, padding: '20px' }}>
                {error && <div style={{ color: 'red', padding: '10px', border: '1px solid red', marginBottom: '20px' }}>{error}</div>}
                {!selectedScenarioId && !error && (
                    <div>
                        <h2>Welcome!</h2>
                        <p>Please select a scenario from the left menu to begin.</p>
                    </div>
                )}
                {selectedScenarioId && <ScenarioViewer scenarioId={selectedScenarioId} />}
            </div>
        </div>
    );
}

export default App;