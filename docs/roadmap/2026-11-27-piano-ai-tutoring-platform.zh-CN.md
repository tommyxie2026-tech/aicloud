# 钢琴智能陪练：开源视觉与声音识别训练平台技术选型及实验路线图

> 状态：垂直场景技术方案 / 待实验验证；不是 aicloud 核心架构契约  
> 日期：2026-11-27  
> 范围：数字钢琴与传统钢琴；音频、MIDI、乐谱图片；第一阶段目标为完整智能陪练 MVP  
> 平台归属：`tommyxie2026-tech/aicloud`  
> 原则：先复现和建立可信基线，再自主训练；模型输出必须经过独立评价，不将演示结果视为生产能力。

## 1. 执行摘要

本项目建设一个能够理解乐谱、识别实际演奏、对齐两者并给出可解释练习反馈的钢琴智能陪练系统。第一阶段同时支持：

- 数字钢琴：采集 MIDI 事件，并同步采集设备音频或线路音频；
- 传统钢琴：通过麦克风采集真实声学信号；
- 乐谱：接收图片/PDF，识别并转换为 MusicXML 或内部标准乐谱表示；
- 反馈：定位错音、漏音、多音、节奏和时值偏差，并保存练习历史。

核心技术决策：

1. **双演奏输入路径、统一音乐事件表示**：数字钢琴走 MIDI 解析路径；传统钢琴走 Audio-to-MIDI（AMT）路径；两条路径汇聚到统一事件结构。
2. **乐谱与演奏分离识别**：乐谱图片由 OMR 转换为 MusicXML；演奏由 MIDI 或 AMT 得到事件序列；通过对齐引擎进行比较。
3. **先采用成熟开源基线**：ByteDance Piano Transcription 作为钢琴 AMT 主基线，Spotify Basic Pitch 作为轻量对照，Audiveris 作为 OMR 基线；不在第一阶段从零训练所有模型。
4. **确定性评价优先**：第一版演奏评价使用可解释的音符匹配、时间对齐和规则指标，不先训练主观评分模型。
5. **aicloud 负责训练与执行治理**：Dataset Registry、Training Job、Experiment Tracking、Model Registry、Independent Evaluation、Deployment Profile 等平台能力由 aicloud 提供；钢琴识别与音乐评价逻辑留在垂直应用层。
6. **先建立可信数据闭环**：数字钢琴同步 MIDI 可作为音频识别的参考标签；传统钢琴测试集必须有人工核验的音符标注，不能把模型自己的预测当作 Ground Truth。

第一阶段建议以四周 MVP 验证端到端可行性，之后再开展真实钢琴域适应、模型蒸馏和轻量化。

## 2. 目标、范围与非目标

### 2.1 第一阶段目标

输入一份乐谱和一段钢琴演奏，系统能够：

1. 将乐谱图片/PDF 转换为可解析的音乐结构；
2. 从数字钢琴 MIDI 或传统钢琴音频中获得实际演奏事件；
3. 将演奏事件与乐谱目标音符对齐；
4. 定位错音、漏音、多音、抢拍、拖拍和时值偏差；
5. 输出小节级、音符级和整段练习报告；
6. 保存数据、模型、评价结果和练习历史，以便后续训练和回归测试。

### 2.2 第一阶段范围

| 能力 | 第一阶段 |
| --- | --- |
| 乐谱图片/PDF 输入 | 纳入，先以清晰单页/单谱表或常见钢琴谱为主 |
| OMR 与 MusicXML | 纳入，允许人工校正 |
| 数字钢琴 MIDI 输入 | 纳入，作为高可信事件来源 |
| 数字钢琴音频输入 | 纳入，用于 AMT 交叉验证 |
| 传统钢琴麦克风录音 | 纳入，先离线分析 |
| 音符/节奏/时值评价 | 纳入 |
| 力度与踏板评价 | 记录并试验，初期不作为硬性准确率门槛 |
| 实时低延迟陪练 | 预留接口，不作为四周 MVP 的强制验收项 |
| 手部/指法视觉分析 | 暂不纳入 |
| 艺术表现、音色与情感评分 | 暂不纳入 |
| 从零训练大型多模态模型 | 不纳入 |

### 2.3 非目标与边界

- 不把 OMR、AMT、MIDI 解析、乐谱对齐和教学建议合并成一个不可解释的端到端模型。
- 不把模型预测自动当成训练标签。
- 不把数字钢琴 MIDI 的准确性等同于传统钢琴声学识别准确性。
- 不在没有独立测试和授权核验的情况下发布模型或数据。
- 不把钢琴领域逻辑放入 aicloud 的平台核心。

## 3. 总体技术架构

### 3.1 逻辑数据流

```text
                 Score Image / PDF
                         |
                         v
                   OMR Pipeline
                         |
                         v
                MusicXML / Score IR
                         |
                         +---------------------------+
                                                     |
Digital Piano MIDI ----> MIDI Parser                 |
                                                     v
Digital Piano Audio ---> AMT Pipeline ------> Piano Event IR
                                                     |
Acoustic Piano Audio --> AMT Pipeline ---------------+
                                                     |
                                                     v
                                         Score Alignment Engine
                                                     |
                                                     v
                                      Performance Evaluation Engine
                                                     |
                                                     v
                                   Practice Feedback / History / Report
```

### 3.2 双输入路径

| 项目 | 数字钢琴 | 传统钢琴 |
| --- | --- | --- |
| 原始输入 | MIDI、音频 | 麦克风音频 |
| 音符事件来源 | MIDI 解析 | AMT 模型推断 |
| 起音时间 | MIDI Note On | 音频起音估计 |
| 结束时间 | MIDI Note Off | 音频时值估计 |
| 力度 | MIDI Velocity | 声学力度估计，需校准 |
| 踏板 | MIDI CC 64 等控制器 | 音频推断，可靠性需单独评估 |
| 第一阶段定位 | 高可信参考路径 | 真实声学泛化路径 |

数字钢琴应尽量同时采集 MIDI 和音频。MIDI 是演奏事件参考，音频用于检验 AMT 模型能否还原设备真实输出。线路输出、耳机输出和麦克风采集的声学特征不同，必须记录采集方式。

### 3.3 统一事件表示

OMR 输出的目标乐谱和 MIDI/AMT 输出的实际演奏都映射到统一的音乐事件模型。推荐最小字段：

| 字段 | 类型/含义 |
| --- | --- |
| piece_id | 乐曲标识 |
| performance_id | 单次演奏标识 |
| pitch | MIDI 音高 |
| onset | 相对起始时间，秒 |
| offset | 相对结束时间，秒 |
| velocity | 0–127；无可靠估计时为空 |
| part/staff/voice | 声部、谱表、声部编号 |
| measure | 小节编号 |
| beat | 小节内拍位或分数拍位 |
| hand | 左/右手/未知；不得仅凭音高强行推断 |
| pedal_events | 踏板控制事件 |
| confidence | 模型识别置信度 |
| source | midi / amt / omr / human-corrected |
| provenance | 数据源、模型版本、人工校正记录 |

建议同时保存原始 MIDI、MusicXML、音频和模型输出，不要只保存转换后的事件表。

### 3.4 对齐策略

采用分层方法：

1. 乐谱解析：从 MusicXML 获得目标音符、节拍、小节和声部；
2. 音符候选匹配：结合音高、声部、相邻音符和时间窗口匹配；
3. 时间对齐：先尝试基于音符事件的动态规划；必要时使用 DTW 或节奏特征辅助；
4. 错误分类：区分错音、漏音、多音、重复音和时间偏差；
5. 置信度处理：低置信度 AMT/OMR 结果标记为待复核，不直接生成确定性教学结论。

演奏速度与乐谱速度不同不应直接判为错误。对齐算法需要容忍整体速度变化，同时保留局部节奏偏差。

## 4. 开源模型与技术选型

### 4.1 音频转 MIDI（AMT）

| 项目 | 定位 | 采用策略 |
| --- | --- | --- |
| [ByteDance Piano Transcription](https://github.com/bytedance/piano_transcription) | 钢琴专用复音转录与训练基线 | 主基线；先复现官方环境和评测 |
| [Spotify Basic Pitch](https://github.com/spotify/basic-pitch) | 轻量通用音频转 MIDI | 对照模型；测试CPU/GPU部署与真实录音泛化 |
| [Onsets and Frames](https://magenta.tensorflow.org/onsets-frames) | 钢琴音符起始与持续时间研究方法 | 研究对照，不作为首个交付依赖 |

选型说明：

- ByteDance 项目优先用于钢琴音频识别基线；训练前先检查仓库当前 commit、依赖、权重获取方式和许可证。
- Basic Pitch 作为轻量对照；核对代码许可证与具体预训练权重的使用条件，二者不可混为一谈。
- 不以公开数据集上的指标直接推断家庭传统钢琴录音表现。
- 先离线分段推理，确认准确性和处理耗时后再实现实时流式处理。

### 4.2 乐谱识别（OMR）

| 项目 | 定位 | 采用策略 |
| --- | --- | --- |
| [Audiveris](https://github.com/Audiveris/audiveris) | 光学乐谱识别，输出 MusicXML | 第一阶段 OMR 基线，保留人工校正 |
| [music21](https://web.mit.edu/music21/) | 音乐结构解析和理论工具 | 解析 MusicXML、节拍、调号和音符结构 |

Audiveris 的 AGPL-3.0 许可证需要在集成、网络服务、修改和分发前进行法律审查。若后续作为闭源商业产品的一部分，应先完成许可证兼容性评估。

### 4.3 数据处理与实验工具

| 工具 | 用途 |
| --- | --- |
| [librosa](https://librosa.org/) | 音频加载、频谱与特征处理 |
| [torchaudio](https://pytorch.org/audio/stable/index.html) | PyTorch 音频数据处理 |
| [pretty_midi](https://github.com/craffel/pretty-midi) | MIDI 读写和事件处理 |
| [mido](https://github.com/mido/mido) | MIDI 消息与设备接入 |
| [OpenCV](https://opencv.org/) | 乐谱图像预处理 |
| [PyTorch](https://pytorch.org/) | 模型训练和微调 |
| [DVC](https://dvc.org/) | 数据版本管理 |
| [MLflow](https://mlflow.org/) | 实验记录、模型版本和指标 |
| [Optuna](https://optuna.org/) | 超参数搜索 |
| [ONNX Runtime](https://onnxruntime.ai/) | CPU/GPU 跨设备推理基线 |
| [FastAPI](https://fastapi.tiangolo.com/) | 应用服务接口 |

第一阶段优先建立可复现环境，不要求所有组件立即部署到 Kubernetes。ByteDance、Basic Pitch、Audiveris 应使用隔离环境，避免旧依赖与现代依赖冲突。

### 4.4 许可证与数据权利检查

必须分开记录：

- 代码许可证；
- 预训练权重许可证；
- 训练数据许可证；
- 测试数据许可证；
- 乐谱图片/出版物版权；
- 用户录音和 MIDI 的授权与保存范围。

MAESTRO 官方数据页列出 CC BY-NC-SA 4.0 条件。它适合符合该许可要求的研究用途，不能默认用于商业训练、商业数据分发或不兼容的衍生数据发布。ByteDance 仓库采用的 MAESTRO 版本与官方当前数据版本也需区分：复现其原始实验时先按仓库要求使用 MAESTRO V2；后续扩展 V3 应单独建立数据版本和实验记录。

## 5. 数据采集、标注与管理

### 5.1 三类数据集

**A. 公开基准集**

- 首轮复现按 ByteDance 仓库要求使用 MAESTRO V2；
- 后续可独立评估 MAESTRO V3；
- 保留官方 train/validation/test 划分；
- 禁止跨划分泄漏同一演奏、录音或近重复片段；
- 记录数据版本、来源、下载日期和许可证。

参考：[MAESTRO 官方数据集页面](https://magenta.tensorflow.org/datasets/maestro)。

**B. 数字钢琴自采数据**

采集 MIDI、音频和对应乐谱。实验覆盖：

- 单音、双音、和弦、复音；
- 低、中、高音区；
- 慢速、正常速度、快速；
- 不同力度；
- 连奏、断奏、重复音；
- 踏板开关与变化；
- 不同难度片段；
- 不同数字钢琴型号和音色设置。

重点采集可控错误样本：漏音、错音、多音、重复音、抢拍、拖拍、时值偏差和踏板变化。

**C. 传统钢琴真实录音**

采集麦克风音频，记录设备、麦克风位置、距离、房间、采样率、声道、演奏条件和环境噪声。至少建立一批人工核验的音符事件标注，作为独立测试集。

### 5.2 采集格式

建议保存原始无损音频（WAV/PCM），采样率可按设备支持选择 44.1kHz 或 48kHz；记录位深、声道和采集路径。不要在采集阶段对原始文件进行有损压缩或不可逆降噪。

推荐目录：

```text
datasets/
  maestro/
    v2/
    v3/
  digital_piano/
    raw_audio/
    raw_midi/
    score/
    annotations/
  acoustic_piano/
    raw_audio/
    score/
    annotations/
  manifests/
  splits/
```

### 5.3 样本清单

每个练习片段使用唯一 ID，建议包含：

- session_id、piece_id、performer_id（匿名化）；
- instrument_type、device_model、audio_capture_path；
- audio_file、midi_file、score_image、musicxml_file；
- sample_rate、channels、bit_depth；
- sync_calibration_id、measured_latency；
- annotation_version、annotator、review_status；
- dataset_split、license、consent_scope。

### 5.4 音频与 MIDI 同步校准

数字钢琴 MIDI 与音频通常不共享完全相同的时钟。音源延迟、音频缓冲、设备输出和录音链路都会引入偏差。

正式采集前：

1. 设计可重复的校准音符序列；
2. 同时记录 MIDI Note On/Off 和音频；
3. 测量 MIDI 事件与音频起音的延迟；
4. 多次重复并统计平均延迟和波动；
5. 分别校准线路输出、耳机输出和麦克风路径；
6. 将校准结果绑定到设备和采集配置。

未校准的同步误差不得计入 AMT 模型的起音误差。

### 5.5 标签可信度

| 数据来源 | 标签可信度 | 用途 |
| --- | --- | --- |
| 数字钢琴原始 MIDI | 高，但需核验设备事件和时钟 | 音频识别参考、演奏事件 |
| 人工校正 MusicXML | 高 | 乐谱标准答案 |
| 传统钢琴人工核验转录 | 高 | 真实声学测试 |
| AMT 模型输出 | 未验证预测 | 推理结果或弱标签 |
| 未复核 OMR 输出 | 未验证预测 | 待校正乐谱 |

模型预测不得直接回流成为可信训练标签。伪标签需要经过置信度筛选、抽样人工复核和独立评价。

## 6. 标准事件结构与评价算法

### 6.1 Piano Event IR

第一版统一事件字段：

```json
{
  "piece_id": "PIECE_001",
  "performance_id": "PRACTICE_000001",
  "events": [
    {
      "pitch": 60,
      "onset": 1.250,
      "offset": 1.750,
      "velocity": 85,
      "part": "piano",
      "staff": "treble",
      "voice": "1",
      "measure": 1,
      "beat": 1.0,
      "hand": "unknown",
      "confidence": 0.98,
      "source": "midi"
    }
  ],
  "pedal_events": [
    {
      "controller": 64,
      "time": 1.500,
      "value": 127
    }
  ]
}
```

约束：

- 时间统一为相对演奏起始点的秒数；
- 保留 MIDI tick/tempo 原始信息或可逆映射；
- hand 允许 unknown，不能简单按音高切分左右手；
- confidence 和 source 必须保留；
- 人工校正需有版本和审计记录；
- 原始 MIDI、MusicXML 和音频不可被标准化结果覆盖。

### 6.2 对齐算法

第一阶段使用可解释的确定性算法：

1. MusicXML 解析为目标音符事件；
2. MIDI/AMT 解析为实际音符事件；
3. 基于音高、声部、局部顺序和时间窗口生成候选匹配；
4. 使用动态规划进行音符序列匹配；
5. 需要时使用 DTW 处理速度变化；
6. 输出音符匹配、未匹配目标和未匹配演奏事件；
7. 对低置信度结果标记 review_required。

不应只使用纯 DTW：音符错漏、多声部和重复音需要事件级匹配规则。

### 6.3 演奏评价

| 评价项 | 第一版方法 |
| --- | --- |
| 错音 | 匹配事件的 pitch 不一致 |
| 漏音 | 目标事件未匹配 |
| 多音 | 实际事件未匹配 |
| 起音偏差 | 实际 onset 与目标 onset 的差值 |
| 时值偏差 | 实际 offset/onset 与目标时值比较 |
| 节奏偏差 | 考虑整体速度后的局部时间偏差 |
| 力度偏差 | 数字钢琴 Velocity 优先；声学估计单独报告 |
| 踏板 | MIDI CC 优先；音频推断标注为低置信度 |

系统需要区分“模型识别错误”和“学生演奏错误”。数字钢琴场景可用 MIDI 与音频互相核验；传统钢琴场景必须保留模型不确定性，避免将识别错误直接反馈给练习者。

## 7. 训练与部署实验环境

### 7.1 推荐实验栈

| 层 | 技术 |
| --- | --- |
| 训练 | PyTorch |
| 音频 | librosa、torchaudio |
| OMR | Audiveris |
| MIDI | pretty_midi、mido |
| 音乐结构 | music21 |
| 数据版本 | DVC |
| 实验跟踪 | MLflow |
| 参数优化 | Optuna |
| 推理 | ONNX Runtime |
| NVIDIA 优化 | 后续评估 TensorRT |
| API | FastAPI |

### 7.2 GPU 资源

- 数据处理、MIDI 解析、MusicXML 解析：CPU 优先；
- OMR 初期可先使用 CPU；
- ByteDance 原始训练配置应以官方仓库为准；既有文档给出的参考为 V100 32GB、Batch Size 12，实际显存取决于版本、输入裁剪、精度和 batch；
- 微调从单 GPU 开始，按显存和吞吐调整 batch；
- 不在没有实测前承诺固定 GPU 数量或 tokens/sec；
- 第一阶段不需要分布式训练。

### 7.3 环境隔离

ByteDance Piano Transcription、Basic Pitch、Audiveris 依赖不同，分别采用独立容器或环境。先复现原始环境，再做依赖升级；迁移后必须用固定样本和评价脚本做回归比较。

## 8. 四周 MVP 实施计划

### 第 1 周：数据和环境

任务：

- 下载并校验 MAESTRO V2；
- 按官方仓库要求复现 ByteDance 环境；
- 安装 Basic Pitch；
- 建立数字钢琴 MIDI 接入；
- 完成音频采集和同步校准；
- 建立统一目录、manifest 和数据版本。

交付：

- 可复现训练/推理环境；
- MAESTRO 数据集版本记录；
- 数字钢琴 WAV/MIDI 样本；
- 数据采集说明。

### 第 2 周：AMT 基线

任务：

- 运行 ByteDance 模型；
- 运行 Basic Pitch；
- 对 MAESTRO 测试集进行评估；
- 对数字钢琴自采音频进行推理；
- 以 MIDI 作为数字钢琴音频的参考；
- 分析音符、起音、结束时间错误。

交付：

- 两种模型基线报告；
- 设备级误差报告；
- 失败案例集。

### 第 3 周：OMR 和事件对齐

任务：

- 使用 Audiveris 将乐谱图片转为 MusicXML；
- 对 MusicXML 做人工抽样校验；
- 使用 music21 解析目标音符；
- 解析实际 MIDI/AMT 事件；
- 实现音符匹配、动态规划和时间对齐；
- 生成错音、漏音、多音和节奏偏差结果。

交付：

- MusicXML 处理流水线；
- Piano Event IR；
- Score Alignment Engine；
- 评价引擎单元测试。

### 第 4 周：端到端陪练原型

任务：

- 上传乐谱；
- 选择 MIDI 或音频演奏输入；
- 运行识别和对齐；
- 展示错误音符和小节位置；
- 生成练习报告；
- 保存练习历史和模型版本。

交付：

- 第一版钢琴智能陪练原型；
- 端到端测试报告；
- 已知问题清单；
- 第二阶段训练任务列表。

### 8.1 实验矩阵

| ID | 数据 | 方法 | 目的 |
| --- | --- | --- | --- |
| E01 | MAESTRO V2 | ByteDance | 复现钢琴 AMT |
| E02 | MAESTRO V2 | Basic Pitch | 对照基线 |
| E03 | 数字钢琴音频 + MIDI | 两种 AMT | 真实设备识别 |
| E04 | MusicXML + MIDI | 事件匹配 | 演奏评价 |
| E05 | 传统钢琴录音 | ByteDance | 声学域泛化 |
| E06 | 传统钢琴录音 | Basic Pitch | 模型对照 |
| E07 | 传统钢琴人工核验标注 | 两种 AMT | 真实录音独立评估 |

E05–E07 的传统钢琴测试集必须有人工核验的音符标注；不得把模型预测当作 Ground Truth。

## 9. 评价指标与验收

### 9.1 AMT 指标

| 指标 | 含义 |
| --- | --- |
| Note Precision / Recall / F1 | 音符识别 |
| Onset Precision / Recall / F1 | 起音检测 |
| Offset F1 | 结束时间检测 |
| Velocity MAE | 力度估计误差 |
| Pedal F1 | 踏板事件识别 |

起音容差可将 50ms 作为一组初始实验参数，同时报告不同容差下的结果。最终评价应遵循基线仓库的指标定义，避免不同评测脚本造成不可比。

### 9.2 演奏评价指标

- Pitch Error Rate；
- Missing Note Rate；
- Extra Note Rate；
- Onset Deviation；
- Duration Deviation；
- Tempo Deviation；
- Measure-level localization accuracy。

### 9.3 工程指标

- 单分钟音频处理时间；
- 实时处理延迟（后续阶段）；
- CPU/GPU 占用和峰值显存；
- 连续录音稳定性；
- 设备间泛化；
- 低置信度结果比例；
- 人工复核比例。

### 9.4 验收标准

1. ByteDance 与 Basic Pitch 至少一个可稳定完成基线推理；
2. MAESTRO 数据版本和官方划分可追溯；
3. 数字钢琴 MIDI 与音频可同步采集；
4. OMR 能输出 MusicXML，且支持人工校正；
5. 三类输入能进入统一事件表示；
6. 能够对齐目标乐谱与实际演奏；
7. 能定位错音、漏音、多音和时间偏差；
8. 能生成可追溯练习报告；
9. 传统钢琴测试集有人工核验标签；
10. 模型、数据、实验和评价结果均有版本记录。

第一阶段先建立基线，不预设未经验证的识别准确率目标。建立基线后，再定义分设备、分难度的性能门槛。

## 10. 第二阶段：自主训练与泛化

第一阶段闭环完成后，按以下顺序开展：

### 10.1 钢琴专用小模型

- 以 ByteDance 模型为 Teacher；
- 以数字钢琴音频/MIDI 配对数据扩充训练；
- 研究微调、蒸馏和模型压缩；
- 比较 FP32、FP16、INT8 等推理配置；
- 同时评估准确率、延迟、显存和设备兼容性。

### 10.2 真实钢琴域适应

重点研究 MAESTRO 到家庭/教室录音的域差异：

- 麦克风频响与位置；
- 房间混响；
- 钢琴音色与机械噪声；
- 背景噪声；
- 演奏水平和速度；
- 不同录音链路。

可研究数据增强、域适应和小规模真实数据微调。必须保留独立真实录音测试集。

### 10.3 视觉与声音交叉验证

- 乐谱视觉识别给出目标音符；
- 音频模型给出演奏估计；
- 数字钢琴 MIDI 提供参考事件；
- 对不一致结果执行置信度分析和复核。

这比第一阶段直接训练一个大型多模态模型更容易验证，也更符合可控小模型训练目标。

### 10.4 实时陪练

在离线识别准确性达标后再实现：

- 音频分帧和缓存；
- 流式 AMT；
- 当前小节定位；
- 增量式对齐；
- 错误提示节流；
- 断音、重复音和踏板状态管理。

## 11. aicloud 平台集成边界

本项目作为 aicloud 的垂直 AI 训练工作负载和参考应用，不改变平台核心产品方向。

| aicloud 能力 | 钢琴项目中的职责 |
| --- | --- |
| Dataset Registry | MAESTRO、数字钢琴、传统钢琴、乐谱数据版本 |
| Training Planner | 训练/微调配置和资源规划 |
| Job Executor | 训练、预处理和批量评估任务执行 |
| Experiment Tracker | 参数、指标、日志和产物 |
| Model Registry | AMT、OMR 模型及版本 |
| Independent Evaluator | 独立测试、回归和模型准入 |
| Deployment Profile | CPU/GPU、边缘设备部署规格 |
| Task Router | 按音频、MIDI、乐谱任务选择处理组件 |
| Trajectory | 数据、模型、实验、评价和反馈的来源链 |
| Continuous Learning | 经审核的新样本进入后续训练 |

### 11.1 推荐工作流

```text
User Data
  |
  v
Data Validation + Consent + Provenance
  |
  v
Dataset Registry
  |
  v
Training / Inference Job
  |
  v
Artifact + Experiment Record
  |
  v
Independent Evaluation
  |
  v
Policy / License / Quality Gate
  |
  v
Model Registry
  |
  v
Deployment Profile
  |
  v
Piano Practice Application
  |
  v
Reviewed Feedback -> Next Dataset Version
```

### 11.2 平台与垂直应用边界

**aicloud 平台负责：**

- 任务定义、执行和状态管理；
- 数据集与模型版本；
- 资源调度和运行环境；
- 实验记录、产物和证据；
- 独立评价和准入；
- 权限、租户、数据驻留和授权治理；
- 失败恢复、审计和回滚。

**钢琴垂直应用负责：**

- 乐谱语义和音乐理论；
- OMR/AMT 业务流水线；
- Piano Event IR；
- Score Alignment；
- 音符、节奏、力度和踏板评价；
- 练习报告和教学交互。

LLM 可以辅助解释练习报告和生成练习建议，但不能替代确定性音符匹配，也不能把模型的自然语言结论当作独立评价证据。

## 12. 风险与控制

| 风险 | 控制措施 |
| --- | --- |
| OMR 误识别 | MusicXML 抽样校验和人工校正 |
| AMT 漏音/错音 | MIDI 参考、置信度和人工核验 |
| MIDI/音频不同步 | 设备级延迟校准 |
| 数据集泄漏 | 按演奏/作品分组划分 |
| MAESTRO 许可限制 | 非商业研究边界审查 |
| OMR AGPL 许可 | 商业集成前法律审查 |
| 模型代码与权重许可不同 | 分开记录许可证和来源 |
| 伪标签污染 | 人工抽样复核和独立评估 |
| 设备域偏差 | 分设备报告和真实录音测试 |
| 评分误导 | 区分识别错误与学生演奏错误 |
| 用户隐私 | 明确录音授权、保留周期和删除机制 |

## 13. 后续里程碑

| 阶段 | 目标 | 主要交付 |
| --- | --- | --- |
| P0 / 4周 | 端到端 MVP | OMR + AMT/MIDI + 对齐 + 练习报告 |
| P1 / 4周 | 真实数据扩展 | 数字/传统钢琴测试集和失败案例 |
| P2 / 4周 | 自主训练 | 钢琴专用微调/蒸馏实验 |
| P3 | 实时陪练 | 流式识别、增量对齐和实时反馈 |
| P4 | 个性化教学 | 历史错误建模、练习推荐和效果评价 |

## 14. 第一阶段交付物

1. 可复现的音频识别环境；
2. ByteDance 与 Basic Pitch 基线报告；
3. 数字钢琴 MIDI/音频采集与同步校准工具；
4. 传统钢琴录音和人工标注测试集；
5. OMR 到 MusicXML 的处理流程；
6. Piano Event IR；
7. Score Alignment Engine；
8. Performance Evaluation Engine；
9. 钢琴智能陪练 MVP；
10. 数据、模型、实验和评价的版本追溯记录。

## 15. 参考资料

- [ByteDance Piano Transcription](https://github.com/bytedance/piano_transcription)
- [Spotify Basic Pitch](https://github.com/spotify/basic-pitch)
- [Google Magenta MAESTRO Dataset](https://magenta.tensorflow.org/datasets/maestro)
- [Google Magenta Onsets and Frames](https://magenta.tensorflow.org/onsets-frames)
- [Audiveris](https://github.com/Audiveris/audiveris)
- [music21](https://web.mit.edu/music21/)
- [pretty_midi](https://github.com/craffel/pretty-midi)
- [mido](https://github.com/mido/mido)
- [librosa](https://librosa.org/)
- [PyTorch Audio](https://pytorch.org/audio/stable/index.html)
- [DVC](https://dvc.org/)
- [MLflow](https://mlflow.org/)
- [ONNX Runtime](https://onnxruntime.ai/)

---

## 16. 决策记录

本文件是垂直场景的技术选型和实验路线图，状态为待实验验证。它不代表上述模型已经在 aicloud 中集成，也不代表训练、数据采集、实时推理或生产部署已经完成。

在进入实现前，需通过实验确认：

1. ByteDance 训练环境和权重可用性；
2. MAESTRO 对应版本及许可边界；
3. 数字钢琴 MIDI/音频同步误差；
4. Audiveris 对目标乐谱类型的识别效果；
5. AMT 在传统钢琴真实录音上的泛化表现；
6. 第一阶段评价指标和可接受阈值。

平台侧后续实现应独立拆分 Dataset、Training Job、Experiment、Model Artifact 和 Evaluation Evidence，不将单个钢琴项目的业务字段直接固化为 aicloud 核心 API。
