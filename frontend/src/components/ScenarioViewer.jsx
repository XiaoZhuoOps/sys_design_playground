import React, { useState, useEffect } from 'react';
import axios from 'axios';

const apiClient = axios.create({
    baseURL: '/api',
});

const ScenarioViewer = ({ scenarioId }) => {
    const [scenario, setScenario] = useState(null);
    const [state, setState] = useState(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(null);
    const [actionLoading, setActionLoading] = useState(false);

    // Fetch scenario details and initial state
    useEffect(() => {
        if (!scenarioId) return;
        setLoading(true);
        Promise.all([
            apiClient.get(`/scenarios/${scenarioId}`),
            apiClient.get(`/scenarios/${scenarioId}/state`)
        ]).then(([scenarioRes, stateRes]) => {
            setScenario(scenarioRes.data);
            setState(stateRes.data);
            setLoading(false);
        }).catch(err => {
            console.error("Failed to fetch scenario data:", err);
            setError("Could not load scenario data.");
            setLoading(false);
        });
    }, [scenarioId]);

    // Poll for state updates
    useEffect(() => {
        if (!scenarioId) return;
        const interval = setInterval(() => {
            apiClient.get(`/scenarios/${scenarioId}/state`)
                .then(res => setState(res.data))
                .catch(err => console.error("State poll failed:", err));
        }, 2000); // Poll every 2 seconds
        return () => clearInterval(interval);
    }, [scenarioId]);

    const handleActionClick = (actionId) => {
        setActionLoading(true);
        apiClient.post(`/scenarios/${scenarioId}/actions/${actionId}`)
            .then(() => {
                // State will update on the next poll
                setActionLoading(false);
            })
            .catch(err => {
                console.error(`Action ${actionId} failed:`, err);
                setActionLoading(false);
            });
    };

    if (!scenarioId) return null;
    if (loading) return <div>Loading...</div>;
    if (error) return <div style={{ color: 'red', padding: '10px', border: '1px solid red' }}>{error}</div>;

    return (
        <div>
            <h2>{scenario.title}</h2>
            <p>{scenario.problem_description}</p>
            <h3>Solution</h3>
            <p>{scenario.solution_description}</p>

            <div style={{ display: 'flex', gap: '20px' }}>
                <div style={{ flex: 1 }}>
                    <h4>Actions</h4>
                    <div>
                        {scenario.actions.map(action => (
                            <button
                                key={action.id}
                                onClick={() => handleActionClick(action.id)}
                                disabled={actionLoading}
                                style={{
                                    marginRight: '10px',
                                    marginBottom: '10px',
                                    padding: '8px 16px',
                                    border: '1px solid #ddd',
                                    backgroundColor: 'white',
                                    cursor: actionLoading ? 'not-allowed' : 'pointer'
                                }}
                            >
                                {action.name}
                            </button>
                        ))}
                    </div>
                </div>
                <div style={{ flex: 1 }}>
                    <h4>Live Dashboard</h4>
                    {state && Object.entries(state).map(([key, value]) => (
                        <div key={key} style={{ marginBottom: '16px', padding: '10px', border: '1px solid #ddd' }}>
                            <strong>{key}:</strong>
                            <pre style={{ margin: '5px 0 0 0', whiteSpace: 'pre-wrap', wordBreak: 'break-all', fontSize: '12px' }}>
                                {typeof value === 'object' ? JSON.stringify(value, null, 2) : value}
                            </pre>
                        </div>
                    ))}
                </div>
            </div>
        </div>
    );
};

export default ScenarioViewer;