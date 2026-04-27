import React, { useEffect, useMemo, useState } from 'react';
import { DragDropContext, Droppable, Draggable } from '@hello-pangea/dnd';
import { useRoadmap } from '../hooks/useRoadmap';
import { useAuth } from '../hooks/useAuth';
import MilestoneCard from './MilestoneCard';
import EpicCard from './EpicCard';
import CreateMilestoneModal from './modals/CreateMilestoneModal';
import CreateEpicModal from './modals/CreateEpicModal';
import UpdateMilestoneModal from './modals/UpdateMilestoneModal';
import EpicMoveModal from './modals/EpicMoveModal';
import UpdateEpicModal from './modals/UpdateEpicModal';
import { Plus, RefreshCw, GripVertical, Compass } from 'lucide-react';
import {
  saveSelectedQuarters,
  loadSelectedQuarters,
  validateQuarterSelection,
  getDefaultQuarters,
} from '../utils/quarterStorage';
import { sortQuarters, validateAndSortSelectedQuarters } from '../utils/sortingUtils';
import './KanbanBoard.css';

const MAX_QUARTERS = 4;

const KanbanBoard = () => {
  const { onProjectChange } = useAuth();
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

  useEffect(() => {
    if (roadmapData?.quarters && selectedQuarters.length === 0) {
      const storedQuarters = loadSelectedQuarters(roadmapData.quarters);
      if (storedQuarters && storedQuarters.length > 0) {
        setSelectedQuarters(storedQuarters);
      } else {
        const defaultQuarters = getDefaultQuarters(roadmapData.quarters);
        setSelectedQuarters(defaultQuarters);
        saveSelectedQuarters(defaultQuarters);
      }
    }
  }, [roadmapData?.quarters, selectedQuarters.length]);

  useEffect(() => {
    if (roadmapData?.quarters && selectedQuarters.length > 0) {
      const validatedQuarters = validateQuarterSelection(selectedQuarters, roadmapData.quarters);
      if (JSON.stringify(validatedQuarters) !== JSON.stringify(selectedQuarters)) {
        setSelectedQuarters(validatedQuarters);
        saveSelectedQuarters(validatedQuarters);
      }
    }
  }, [roadmapData?.quarters, selectedQuarters]);

  useEffect(() => {
    const unsubscribe = onProjectChange((newProject, prevProject) => {
      if (newProject !== prevProject) loadRoadmap();
    });
    return () => unsubscribe();
  }, [onProjectChange, loadRoadmap]);

  const handleDragEnd = async (result) => {
    const { destination, source, draggableId } = result;
    if (!destination) return;
    if (
      destination.droppableId === source.droppableId &&
      destination.index === source.index
    ) return;
    const newMilestoneId = destination.droppableId.replace('milestone-', '');
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
    let next;
    if (isSelected) {
      next = selectedQuarters.filter(q => q !== quarter);
    } else if (selectedQuarters.length < MAX_QUARTERS) {
      next = [...selectedQuarters, quarter];
    } else {
      next = [...selectedQuarters.slice(1), quarter];
    }
    const sorted = sortQuarters(next);
    setSelectedQuarters(sorted);
    saveSelectedQuarters(sorted);
  };

  const sortedQuarters = useMemo(
    () => sortQuarters(roadmapData?.quarters || []),
    [roadmapData?.quarters]
  );
  const sortedSelectedQuarters = useMemo(
    () => validateAndSortSelectedQuarters(selectedQuarters, sortedQuarters),
    [selectedQuarters, sortedQuarters]
  );

  if (isLoading) {
    return (
      <div className="kanban-state">
        <div className="atlas-spinner" />
        <p className="serif kanban-state__title">Drawing the chart…</p>
        <p className="kanban-state__sub">Pillars, milestones &amp; epics, fetched from Jira.</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="kanban-state kanban-state--error">
        <Compass size={28} strokeWidth={1.5} />
        <h3 className="serif kanban-state__title">Lost the trail</h3>
        <p className="kanban-state__sub">{error}</p>
        <button onClick={loadRoadmap} className="btn btn-primary">
          <RefreshCw size={14} strokeWidth={1.75} />
          Try again
        </button>
      </div>
    );
  }

  if (!roadmapData || !roadmapData.pillars) {
    return (
      <div className="kanban-state">
        <Compass size={28} strokeWidth={1.5} />
        <h3 className="serif kanban-state__title">No pillars yet</h3>
        <p className="kanban-state__sub">Configure pillars in Jira and refresh.</p>
        <button onClick={loadRoadmap} className="btn btn-primary">
          <RefreshCw size={14} strokeWidth={1.75} />
          Refresh
        </button>
      </div>
    );
  }

  const { pillars, quarters } = roadmapData;
  const displayQuarters = sortedSelectedQuarters.length > 0 ? sortedSelectedQuarters : sortedQuarters;
  const totalEpics = pillars.reduce(
    (acc, p) => acc + (p.milestones || []).reduce((s, m) => s + (m.epics?.length || 0), 0),
    0
  );
  const totalMilestones = pillars.reduce((acc, p) => acc + (p.milestones || []).length, 0);

  return (
    <div className="kanban-container fade-in">
      {/* Section masthead */}
      <div className="kanban-masthead">
        <div className="kanban-masthead__head">
          <span className="kanban-masthead__chapter mono">CHAPTER 01</span>
          <h2 className="kanban-masthead__title">
            <span className="serif">The </span>
            <span>Roadmap</span>
            <span className="serif"> — by quarter, by pillar.</span>
          </h2>
          <p className="kanban-masthead__sub">
            Drag epics across milestones; reshape the future without leaving the page.
          </p>
        </div>
        <dl className="kanban-stats">
          <div className="kanban-stat">
            <dt>Pillars</dt>
            <dd className="mono tnum">{String(pillars.length).padStart(2, '0')}</dd>
          </div>
          <div className="kanban-stat">
            <dt>Milestones</dt>
            <dd className="mono tnum">{String(totalMilestones).padStart(2, '0')}</dd>
          </div>
          <div className="kanban-stat">
            <dt>Epics</dt>
            <dd className="mono tnum">{String(totalEpics).padStart(2, '0')}</dd>
          </div>
        </dl>
      </div>

      {/* Controls */}
      <div className="controls-bar">
        <div className="controls-bar__group">
          <span className="controls-bar__label">Quarters in view</span>
          <div className="quarter-chips">
            {sortedQuarters.length === 0 && (
              <span className="quarter-empty">No quarters available</span>
            )}
            {sortedQuarters.map((quarter) => {
              const active = selectedQuarters.includes(quarter);
              const disabled = !active && selectedQuarters.length >= MAX_QUARTERS;
              return (
                <button
                  key={quarter}
                  type="button"
                  onClick={() => handleQuarterToggle(quarter)}
                  disabled={disabled}
                  className={`quarter-chip mono${active ? ' is-active' : ''}${disabled ? ' is-disabled' : ''}`}
                  aria-pressed={active}
                >
                  {quarter}
                </button>
              );
            })}
          </div>
          <span className="controls-bar__hint">{selectedQuarters.length}/{MAX_QUARTERS}</span>
        </div>
        <button onClick={loadRoadmap} className="btn btn-sm btn-ghost" title="Refresh data">
          <RefreshCw size={13} strokeWidth={1.75} />
          Refresh
        </button>
      </div>

      {/* Kanban Board */}
      <DragDropContext onDragEnd={handleDragEnd}>
        <div
          className="kanban-board"
          style={{ '--col-count': displayQuarters.length || 1 }}
        >
          <div className="kanban-quarters-header">
            <div className="kanban-pillar-header">
              <span className="serif kanban-pillar-header__label">Pillars</span>
              <span className="mono kanban-pillar-header__rule">/ rows</span>
            </div>
            {displayQuarters.map((quarter) => (
              <div key={quarter} className="kanban-quarter-header">
                <span className="mono">{quarter}</span>
              </div>
            ))}
          </div>

          {pillars.map((pillar, idx) => (
            <div key={pillar.id} className="kanban-pillar-row">
              <div className="kanban-pillar-info">
                <div className="kanban-pillar-info__num mono">{String(idx + 1).padStart(2, '0')}</div>
                <div className="kanban-pillar-info__main">
                  <h3 className="kanban-pillar-info__name">{pillar.name}</h3>
                  <span className="kanban-pillar-info__key mono">{pillar.key}</span>
                  {pillar.component && (
                    <span className="kanban-pillar-info__component">{pillar.component}</span>
                  )}
                </div>
                <button
                  onClick={() => handleCreateMilestone(pillar)}
                  className="btn btn-sm btn-icon btn-ghost kanban-pillar-info__add"
                  title="Create milestone"
                  aria-label={`Create milestone in ${pillar.name}`}
                >
                  <Plus size={14} strokeWidth={1.75} />
                </button>
              </div>

              {displayQuarters.map((quarter) => {
                const milestones = (pillar.milestones || []).filter(m => m.quarter === quarter);
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
                                    className={`epics-container${snapshot.isDraggingOver ? ' is-drag-over' : ''}`}
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
                                            className={`epic-draggable${snapshot.isDragging ? ' is-dragging' : ''}`}
                                          >
                                            <div className={`epic-card-with-handle${snapshot.isDragging ? ' is-dragging' : ''}`}>
                                              <div
                                                className="epic-drag-handle"
                                                {...provided.dragHandleProps}
                                                title="Drag to move epic"
                                              >
                                                <GripVertical size={12} strokeWidth={1.5} />
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
                                className="create-epic-btn"
                                title="Add epic to milestone"
                              >
                                <Plus size={12} strokeWidth={1.75} />
                                Add epic
                              </button>
                            </div>
                          </div>
                        ))}
                      </div>
                    ) : (
                      <button
                        onClick={() => handleCreateMilestone(pillar, quarter)}
                        className="empty-quarter"
                        title={`Create a milestone in ${quarter}`}
                      >
                        <Plus size={14} strokeWidth={1.75} />
                        <span>Add milestone</span>
                        <span className="empty-quarter__hint mono">{quarter}</span>
                      </button>
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
          availableMilestones={(roadmapData?.pillars || []).flatMap(p => p.milestones || [])}
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
