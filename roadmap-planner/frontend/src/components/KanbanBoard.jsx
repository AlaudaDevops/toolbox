import React, { useState } from 'react';
import { DragDropContext, Droppable, Draggable } from 'react-beautiful-dnd';
import { useRoadmap } from '../hooks/useRoadmap';
import MilestoneCard from './MilestoneCard';
import EpicCard from './EpicCard';
import CreateMilestoneModal from './modals/CreateMilestoneModal';
import CreateEpicModal from './modals/CreateEpicModal';
import UpdateMilestoneModal from './modals/UpdateMilestoneModal';
import EpicMoveModal from './modals/EpicMoveModal';
import UpdateEpicModal from './modals/UpdateEpicModal';
import { Plus, RefreshCw, GripVertical } from 'lucide-react';
import {
  saveSelectedQuarters,
  loadSelectedQuarters,
  validateQuarterSelection,
  getDefaultQuarters
} from '../utils/quarterStorage';
import { sortQuarters, validateAndSortSelectedQuarters } from '../utils/sortingUtils';
import './KanbanBoard.css';

const KanbanBoard = () => {
  const { roadmapData, isLoading, error, loadRoadmap, moveEpic } = useRoadmap();
  const [showCreateMilestone, setShowCreateMilestone] = useState(false);
  const [showCreateEpic, setShowCreateEpic] = useState(false);
  const [showUpdateMilestone, setShowUpdateMilestone] = useState(false);
  const [showEpicMove, setShowEpicMove] = useState(false);
  const [showUpdateEpic, setShowUpdateEpic] = useState(false);
  const [selectedPillar, setSelectedPillar] = useState(null);
  const [selectedMilestone, setSelectedMilestone] = useState(null);
  const [selectedEpic, setSelectedEpic] = useState(null);
  const [selectedQuarter, setSelectedQuarter] = useState(null);
  const [selectedQuarters, setSelectedQuarters] = useState([]);

  // Initialize selected quarters when roadmap data loads
  React.useEffect(() => {
    if (roadmapData?.quarters && selectedQuarters.length === 0) {
      // Try to load from localStorage first
      const storedQuarters = loadSelectedQuarters(roadmapData.quarters);

      if (storedQuarters && storedQuarters.length > 0) {
        // Use stored quarters if available and valid
        setSelectedQuarters(storedQuarters);
      } else {
        // Fall back to default (first 3 quarters)
        const defaultQuarters = getDefaultQuarters(roadmapData.quarters);
        setSelectedQuarters(defaultQuarters);
        // Save the default selection
        saveSelectedQuarters(defaultQuarters);
      }
    }
  }, [roadmapData?.quarters, selectedQuarters.length]);

  // Validate and update selected quarters when available quarters change
  React.useEffect(() => {
    if (roadmapData?.quarters && selectedQuarters.length > 0) {
      const validatedQuarters = validateQuarterSelection(selectedQuarters, roadmapData.quarters);

      // Only update if the validated selection is different
      if (JSON.stringify(validatedQuarters) !== JSON.stringify(selectedQuarters)) {
        setSelectedQuarters(validatedQuarters);
        saveSelectedQuarters(validatedQuarters);
      }
    }
  }, [roadmapData?.quarters, selectedQuarters]);

  const handleDragEnd = async (result) => {
    const { destination, source, draggableId } = result;

    // If dropped outside a droppable area
    if (!destination) return;

    // If dropped in the same position
    if (
      destination.droppableId === source.droppableId &&
      destination.index === source.index
    ) {
      return;
    }

    // Extract milestone ID from droppable ID (format: "milestone-{id}")
    const newMilestoneId = destination.droppableId.replace('milestone-', '');

    // Move the epic
    await moveEpic(draggableId, newMilestoneId);
  };

  const handleCreateMilestone = (pillar, preselectedQuarter = null) => {
    setSelectedPillar(pillar);
    setSelectedQuarter(preselectedQuarter);
    setShowCreateMilestone(true);
  };

  const handleCreateEpic = (milestone) => {
    setSelectedMilestone(milestone);
    setShowCreateEpic(true);
  };

  const handleUpdateMilestone = (milestone) => {
    setSelectedMilestone(milestone);
    setShowUpdateMilestone(true);
  };

  const handleMoveEpic = (epic, currentMilestone) => {
    setSelectedEpic(epic);
    setSelectedMilestone(currentMilestone);
    setShowEpicMove(true);
  };

  const handleUpdateEpic = (epic) => {
    setSelectedEpic(epic);
    setShowUpdateEpic(true);
  };

  const handleQuarterToggle = (quarter) => {
    if (!roadmapData?.quarters) return;

    const isSelected = selectedQuarters.includes(quarter);
    let newSelectedQuarters;

    if (isSelected) {
      // Remove quarter if it's selected
      newSelectedQuarters = selectedQuarters.filter(q => q !== quarter);
    } else {
      // Add quarter if not selected, but limit to 3 quarters max
      if (selectedQuarters.length < 4) {
        newSelectedQuarters = [...selectedQuarters, quarter];
      } else {
        // Replace the first selected quarter with the new one
        newSelectedQuarters = [...selectedQuarters.slice(1), quarter];
      }
    }

    // Sort the new selection and update state
    const sortedNewSelection = sortQuarters(newSelectedQuarters);
    setSelectedQuarters(sortedNewSelection);
    saveSelectedQuarters(sortedNewSelection);
  };

  if (isLoading) {
    return (
      <div className="kanban-loading">
        <div className="loading-spinner"></div>
        <p>Loading roadmap...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="kanban-error">
        <h3>Error loading roadmap</h3>
        <p>{error}</p>
        <button onClick={loadRoadmap} className="btn btn-primary">
          <RefreshCw size={16} />
          Retry
        </button>
      </div>
    );
  }

  if (!roadmapData || !roadmapData.pillars) {
    return (
      <div className="kanban-empty">
        <h3>No roadmap data available</h3>
        <button onClick={loadRoadmap} className="btn btn-primary">
          <RefreshCw size={16} />
          Refresh
        </button>
      </div>
    );
  }

  const { pillars, quarters } = roadmapData;

  // Debug logging
  console.log('KanbanBoard roadmapData:', roadmapData);
  console.log('Pillars:', pillars);
  console.log('Quarters:', quarters);
  console.log('Selected quarters:', selectedQuarters);

  // Use selected quarters for display, fallback to all quarters if none selected
  // Ensure quarters are sorted chronologically
  const sortedQuarters = sortQuarters(quarters || []);
  const sortedSelectedQuarters = validateAndSortSelectedQuarters(selectedQuarters, sortedQuarters);
  const displayQuarters = sortedSelectedQuarters.length > 0 ? sortedSelectedQuarters : sortedQuarters;

  return (
    <div className="kanban-container">
      {/* Compact Controls Bar */}
      <div className="controls-bar">
        <div className="quarter-selection-compact">
          <span className="quarter-label">Quarters:</span>
          <div className="quarter-checkboxes-compact">
            {sortedQuarters.map((quarter) => (
              <label key={quarter} className="quarter-checkbox-compact">
                <input
                  type="checkbox"
                  checked={selectedQuarters.includes(quarter)}
                  onChange={() => handleQuarterToggle(quarter)}
                  disabled={!selectedQuarters.includes(quarter) && selectedQuarters.length >= 4}
                />
                <span className="quarter-label-compact">{quarter}</span>
              </label>
            ))}
          </div>
          <div><button onClick={loadRoadmap} className="btn btn-sm" title="Refresh data">
            <RefreshCw size={16} />
          </button></div>
        </div>

        {/* Future: Add more controls here like pending epics filter */}
        <div className="future-controls">
          {/* Placeholder for future functionality */}
        </div>
      </div>

      {/* Kanban Board */}
      <DragDropContext onDragEnd={handleDragEnd}>
        <div className="kanban-board">
          {/* Header Row with Quarters */}
          <div className="kanban-quarters-header">
            <div className="kanban-pillar-header">Pillars</div>
            {displayQuarters.map((quarter) => (
              <div key={quarter} className="kanban-quarter-header">
                {quarter}
              </div>
            ))}
          </div>

          {/* Pillar Rows */}
          {pillars.map((pillar) => (
            <div key={pillar.id} className="kanban-pillar-row">
              {/* Pillar Info */}
              <div className="kanban-pillar-info">
                <h3>{pillar.name}</h3>
                <p className="pillar-key">{pillar.key}</p>
                {pillar.component && (
                  <span className="pillar-component">{pillar.component}</span>
                )}
                <button
                  onClick={() => handleCreateMilestone(pillar)}
                  className="btn btn-sm btn-primary"
                  title="Create Milestone"
                >
                  <Plus size={14} />
                </button>
              </div>

              {/* Quarter Columns */}
              {displayQuarters.map((quarter) => {
                const milestones = pillar.milestones.filter(m => m.quarter === quarter);

                return (
                  <div key={`${pillar.id}-${quarter}`} className="kanban-quarter-cell">
                    {milestones.length > 0 ? (
                      <div className="milestone-container">
                        {milestones.map((milestone) => (
                          <div key={milestone.id} className="milestone-wrapper">
                            <MilestoneCard
                              milestone={milestone}
                              onUpdateMilestone={handleUpdateMilestone}
                            />

                            <div className="milestone-epics-section">
                              <Droppable droppableId={`milestone-${milestone.id}`}>
                                {(provided, snapshot) => (
                                  <div
                                    ref={provided.innerRef}
                                    {...provided.droppableProps}
                                    className={`epics-container ${
                                      snapshot.isDraggingOver ? 'drag-over' : ''
                                    }`}
                                  >
                                    {milestone.epics.map((epic, index) => (
                                      <Draggable
                                        key={epic.id}
                                        draggableId={epic.id}
                                        index={index}
                                      >
                                        {(provided, snapshot) => (
                                          <div
                                            ref={provided.innerRef}
                                            {...provided.draggableProps}
                                            className={`epic-draggable ${
                                              snapshot.isDragging ? 'dragging' : ''
                                            }`}
                                          >
                                            <div className={`epic-card-with-handle ${snapshot.isDragging ? 'dragging' : ''}`}>
                                              <div
                                                className="epic-drag-handle"
                                                {...provided.dragHandleProps}
                                                title="Drag to move epic"
                                              >
                                                <GripVertical size={14} />
                                              </div>
                                              <EpicCard
                                                epic={epic}
                                                isDragging={snapshot.isDragging}
                                                onMoveEpic={handleMoveEpic}
                                                onUpdateEpic={handleUpdateEpic}
                                                currentMilestone={milestone}
                                              />
                                            </div>
                                          </div>
                                        )}
                                      </Draggable>
                                    ))}
                                    {provided.placeholder}
                                  </div>
                                )}
                              </Droppable>

                              <button
                                onClick={() => handleCreateEpic(milestone)}
                                className="btn btn-sm btn-outline create-epic-btn"
                                title="Add Epic"
                              >
                                <Plus size={14} />
                                Add Epic
                              </button>
                            </div>
                          </div>
                        ))}
                      </div>
                    ) : (
                      <div className="empty-quarter">
                        <button
                          onClick={() => handleCreateMilestone(pillar, quarter)}
                          className="btn btn-sm btn-secondary create-milestone-btn"
                        >
                          <Plus size={14} />
                          Add Milestone
                        </button>
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
          ))}
        </div>
      </DragDropContext>

      {/* Modals */}
      {showCreateMilestone && (
        <CreateMilestoneModal
          pillar={selectedPillar}
          quarters={quarters || []}
          preselectedQuarter={selectedQuarter}
          onClose={() => {
            setShowCreateMilestone(false);
            setSelectedPillar(null);
            setSelectedQuarter(null);
          }}
        />
      )}

      {showCreateEpic && (
        <CreateEpicModal
          milestone={selectedMilestone}
          onClose={() => {
            setShowCreateEpic(false);
            setSelectedMilestone(null);
          }}
        />
      )}

      {showUpdateMilestone && (
        <UpdateMilestoneModal
          milestone={selectedMilestone}
          quarters={quarters || []}
          onClose={() => {
            setShowUpdateMilestone(false);
            setSelectedMilestone(null);
          }}
        />
      )}

      {showEpicMove && (
        <EpicMoveModal
          epic={selectedEpic}
          currentMilestone={selectedMilestone}
          availableMilestones={roadmapData?.pillars?.flatMap(p => p.milestones) || []}
          onClose={() => {
            setShowEpicMove(false);
            setSelectedEpic(null);
            setSelectedMilestone(null);
          }}
        />
      )}

      {showUpdateEpic && (
        <UpdateEpicModal
          epic={selectedEpic}
          onClose={() => {
            setShowUpdateEpic(false);
            setSelectedEpic(null);
          }}
        />
      )}
    </div>
  );
};

export default KanbanBoard;
